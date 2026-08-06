package engine

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/ilp"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
	"github.com/mirkobrombin/nan-core/pkg/store"
	"github.com/mirkobrombin/nan-core/pkg/vsa"
)

type IngestResult struct {
	Contradiction *belief.Contradiction
}

type Engine struct {
	beliefs  *belief.Store
	graph    *kg.MemGraph
	log      store.Log
	semantic *SemanticIndex

	rules    []rules.Rule
	inducer  ilp.Inducer
	ilpAudit []ilp.Example
	metrics  *Metrics

	derivMu     sync.RWMutex
	derivations map[belief.Atom][]DerivationProof
}

type Option func(*Engine) error

func WithLog(l store.Log) Option {
	return func(e *Engine) error {
		e.log = l
		return nil
	}
}

func WithWAL(w *store.WAL) Option { return WithLog(w) }

func WithInducer(i ilp.Inducer) Option {
	return func(e *Engine) error {
		if i == nil {
			e.inducer = ilp.NoopInducer{}
			return nil
		}
		e.inducer = i
		return nil
	}
}

func New(opts ...Option) (*Engine, error) {
	e := &Engine{
		beliefs:     belief.NewStore(),
		graph:       kg.NewMemGraph(),
		inducer:     ilp.NoopInducer{},
		metrics:     NewMetrics(10000),
		semantic:    newSemanticIndex(vsa.DefaultDims),
		derivations: make(map[belief.Atom][]DerivationProof),
	}
	for _, opt := range opts {
		if err := opt(e); err != nil {
			return nil, err
		}
	}
	return e, nil
}

func (e *Engine) BeliefStore() *belief.Store { return e.beliefs }
func (e *Engine) Graph() *kg.MemGraph        { return e.graph }
func (e *Engine) Metrics() *Metrics          { return e.metrics }
func (e *Engine) Semantic() *SemanticIndex   { return e.semantic }

// CountIncoming counts distinct subjects that satisfy (?x predicate to).
func (e *Engine) CountIncoming(to kg.NodeID, predicate string) int {
	return e.graph.CountIncoming(to, predicate)
}

// RelatedEdges returns sibling atoms: same subject and predicate, different objects.
// Useful for evidence listing in verbalization (e.g. "known values: a, b and c").
func (e *Engine) RelatedEdges(atom belief.Atom) []belief.Atom {
	edges := e.graph.EdgesFrom(atom.From)
	var out []belief.Atom
	for _, edge := range edges {
		if edge.Predicate == atom.Predicate {
			out = append(out, belief.Atom{From: edge.From, Predicate: edge.Predicate, To: edge.To})
		}
	}
	return out
}

// AllEdgesFrom returns every fact whose subject is the given node.
func (e *Engine) AllEdgesFrom(from kg.NodeID) []belief.Atom {
	edges := e.graph.EdgesFrom(from)
	out := make([]belief.Atom, 0, len(edges))
	for _, edge := range edges {
		out = append(out, belief.Atom{From: edge.From, Predicate: edge.Predicate, To: edge.To})
	}
	return out
}

// derivedSourcePrefix marks a belief the engine wrote itself while applying a
// rule. Everything else is a base fact a caller asserted.
const derivedSourcePrefix = "rule:"

// resolvedFalse reports whether an atom is authoritatively resolved to false.
// Such an atom must never enter the graph, or a rule could derive from a
// premise the store holds to be false.
func (e *Engine) resolvedFalse(atom belief.Atom) bool {
	r, ok := e.beliefs.Resolution(atom)
	return ok && r.Polarity == belief.PolarityNegative
}

// Ingest adds a belief to the belief store.
// For positive beliefs it also projects the atom into the KG as a fact, unless
// the atom is authoritatively resolved false.
func (e *Engine) ingestNoLog(b belief.Belief) (*belief.Contradiction, error) {
	c, err := e.beliefs.Add(b)
	if err != nil {
		return nil, err
	}
	if b.Polarity == belief.PolarityPositive && !e.resolvedFalse(b.Atom) {
		_ = e.graph.AddFact(b.Atom.From, b.Atom.Predicate, b.Atom.To)
		e.semantic.Index(b.Atom)
	}
	return c, nil
}

// recompute rebuilds the graph and every derived fact from the base facts and
// the resolutions the store holds. A fact resolved false is not projected, so a
// rule cannot use it as a premise, and any earlier derivation that rested on it
// is dropped rather than left behind. Call it whenever a resolution can flip an
// atom the graph already carries.
func (e *Engine) recompute() {
	var base []belief.Belief
	for _, atom := range e.beliefs.Atoms() {
		for _, b := range e.beliefs.Snapshot(atom) {
			if !strings.HasPrefix(b.Source, derivedSourcePrefix) {
				base = append(base, b)
			}
		}
	}
	resolutions := e.beliefs.Resolutions()

	e.beliefs = belief.NewStore()
	e.graph = kg.NewMemGraph()
	e.semantic = newSemanticIndex(vsa.DefaultDims)
	e.derivMu.Lock()
	e.derivations = make(map[belief.Atom][]DerivationProof)
	e.derivMu.Unlock()

	// Resolutions first, so re-ingesting a base fact respects them and a
	// false-resolved atom is kept out of the graph.
	for _, r := range resolutions {
		_ = e.beliefs.SetResolution(r)
	}
	for _, b := range base {
		_, _ = e.ingestNoLog(b)
	}
	e.ApplyRules(1000)
}

func (e *Engine) Ingest(b belief.Belief) (IngestResult, error) {
	t0 := time.Now()
	c, err := e.ingestNoLog(b)
	if err != nil {
		e.metrics.ErrorCount.Add(1)
		return IngestResult{}, err
	}

	if e.log != nil {
		payload, err := encodeBelief(b)
		if err != nil {
			return IngestResult{}, err
		}
		if err := e.log.Append(store.Record{Type: store.EventFactAdded, Payload: payload}); err != nil {
			return IngestResult{}, err
		}
	}

	e.ilpAudit = append(e.ilpAudit, ilp.Example{Belief: b})
	_ = e.inducer.Observe(ilp.Example{Belief: b})

	e.ApplyRules(1000)
	e.metrics.IngestCount.Add(1)
	e.metrics.Record("", "ingest", time.Since(t0), nil)
	return IngestResult{Contradiction: c}, nil
}

func (e *Engine) Resolve(r belief.Resolution) error {
	t0 := time.Now()
	if err := e.beliefs.SetResolution(r); err != nil {
		e.metrics.ErrorCount.Add(1)
		return err
	}
	if e.log != nil {
		payload, err := encodeResolution(r)
		if err != nil {
			return err
		}
		if err := e.log.Append(store.Record{Type: store.EventResolutionSet, Payload: payload}); err != nil {
			return err
		}
	}
	// A resolution can turn an atom the graph already carries false, so the
	// graph and every derivation are rebuilt rather than only extended.
	e.recompute()
	e.metrics.ResolveCount.Add(1)
	e.metrics.Record("", "resolve", time.Since(t0), nil)
	return nil
}

func (e *Engine) ReplayFromLog() error {
	if e.log == nil {
		return errors.New("log not configured")
	}

	e.beliefs = belief.NewStore()
	e.graph = kg.NewMemGraph()
	e.rules = nil
	e.ilpAudit = nil
	e.derivMu.Lock()
	e.derivations = make(map[belief.Atom][]DerivationProof)
	e.derivMu.Unlock()

	err := e.log.Replay(func(rec store.Record) error {
		switch rec.Type {
		case store.EventFactAdded:
			b, err := decodeBelief(rec.Payload)
			if err != nil {
				return err
			}
			_, err = e.ingestNoLog(b)
			return err
		case store.EventResolutionSet:
			r, err := decodeResolution(rec.Payload)
			if err != nil {
				return err
			}
			return e.beliefs.SetResolution(r)
		case store.EventRuleAdded:
			r, err := decodeRule(rec.Payload)
			if err != nil {
				return err
			}
			e.rules = append(e.rules, r)
			return nil
		default:
			return nil
		}
	})
	if err != nil {
		return err
	}
	// Derived facts are deterministic from (facts + resolutions + rules). The
	// graph is built resolution-aware here, so replaying the fact of an atom
	// later resolved false does not leave a premise behind.
	e.recompute()
	return nil
}

func (e *Engine) ReplayFromWAL() error { return e.ReplayFromLog() }
