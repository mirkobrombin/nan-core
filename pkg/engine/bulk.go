package engine

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/store"
)

// BulkResult summarizes a bulk ingest operation.
type BulkResult struct {
	Total          int
	Ingested       int
	Contradictions int
	Errors         int
	Duration       time.Duration
}

// BulkIngestTriples loads triples from a reader in TSV format: from\tpredicate\tto[\tpolarity]
// Polarity defaults to "+" if omitted. Applies rules once at the end (not per-line).
func (e *Engine) BulkIngestTriples(r io.Reader, source string) BulkResult {
	t0 := time.Now()
	result := BulkResult{}
	sc := bufio.NewScanner(r)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		result.Total++

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			result.Errors++
			continue
		}

		pol := belief.PolarityPositive
		if len(parts) >= 4 && parts[3] == "-" {
			pol = belief.PolarityNegative
		}

		atom := belief.Atom{
			From:      kg.NodeID(strings.TrimSpace(parts[0])),
			Predicate: strings.TrimSpace(parts[1]),
			To:        kg.NodeID(strings.TrimSpace(parts[2])),
		}

		c, err := e.ingestNoLog(belief.Belief{
			Atom:     atom,
			Polarity: pol,
			Source:   source,
		})
		if err != nil {
			result.Errors++
			continue
		}

		// WAL append
		if e.log != nil {
			payload, err := encodeBelief(belief.Belief{Atom: atom, Polarity: pol, Source: source})
			if err == nil {
				_ = e.log.Append(store.Record{Type: store.EventFactAdded, Payload: payload})
			}
		}

		result.Ingested++
		if c != nil {
			result.Contradictions++
		}
	}

	// Apply rules once at the end (batch optimization)
	e.ApplyRules(10000)

	result.Duration = time.Since(t0)
	e.metrics.IngestCount.Add(int64(result.Ingested))
	return result
}

// BulkIngestCSV loads triples from CSV: from,predicate,to[,polarity]
func (e *Engine) BulkIngestCSV(r io.Reader, source string) BulkResult {
	t0 := time.Now()
	result := BulkResult{}
	cr := csv.NewReader(r)

	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors++
			continue
		}
		result.Total++

		if len(record) < 3 {
			result.Errors++
			continue
		}

		pol := belief.PolarityPositive
		if len(record) >= 4 && record[3] == "-" {
			pol = belief.PolarityNegative
		}

		atom := belief.Atom{
			From:      kg.NodeID(strings.TrimSpace(record[0])),
			Predicate: strings.TrimSpace(record[1]),
			To:        kg.NodeID(strings.TrimSpace(record[2])),
		}

		c, err := e.ingestNoLog(belief.Belief{Atom: atom, Polarity: pol, Source: source})
		if err != nil {
			result.Errors++
			continue
		}

		if e.log != nil {
			payload, err := encodeBelief(belief.Belief{Atom: atom, Polarity: pol, Source: source})
			if err == nil {
				_ = e.log.Append(store.Record{Type: store.EventFactAdded, Payload: payload})
			}
		}

		result.Ingested++
		if c != nil {
			result.Contradictions++
		}
	}

	e.ApplyRules(10000)

	result.Duration = time.Since(t0)
	e.metrics.IngestCount.Add(int64(result.Ingested))
	return result
}

// BulkIngestJSON loads an array of triples from JSON: [{"from":"...","predicate":"...","to":"...","polarity":"+/-"}]
func (e *Engine) BulkIngestJSON(r io.Reader, source string) BulkResult {
	t0 := time.Now()
	result := BulkResult{}

	type jsonTriple struct {
		From      string `json:"from"`
		Predicate string `json:"predicate"`
		To        string `json:"to"`
		Polarity  string `json:"polarity"`
	}

	var triples []jsonTriple
	if err := json.NewDecoder(r).Decode(&triples); err != nil {
		result.Errors++
		result.Duration = time.Since(t0)
		return result
	}

	for _, tr := range triples {
		result.Total++
		pol := belief.PolarityPositive
		if tr.Polarity == "-" {
			pol = belief.PolarityNegative
		}

		atom := belief.Atom{
			From:      kg.NodeID(tr.From),
			Predicate: tr.Predicate,
			To:        kg.NodeID(tr.To),
		}

		c, err := e.ingestNoLog(belief.Belief{Atom: atom, Polarity: pol, Source: source})
		if err != nil {
			result.Errors++
			continue
		}

		if e.log != nil {
			payload, err := encodeBelief(belief.Belief{Atom: atom, Polarity: pol, Source: source})
			if err == nil {
				_ = e.log.Append(store.Record{Type: store.EventFactAdded, Payload: payload})
			}
		}

		result.Ingested++
		if c != nil {
			result.Contradictions++
		}
	}

	e.ApplyRules(10000)

	result.Duration = time.Since(t0)
	e.metrics.IngestCount.Add(int64(result.Ingested))
	return result
}

func (r BulkResult) String() string {
	return fmt.Sprintf("total=%d ingested=%d contradictions=%d errors=%d duration=%s",
		r.Total, r.Ingested, r.Contradictions, r.Errors, r.Duration)
}
