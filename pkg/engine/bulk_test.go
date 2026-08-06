package engine

import (
	"strings"
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

func mkTestEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func mkAtom(from, pred, to string) belief.Atom {
	return belief.Atom{From: kg.NodeID(from), Predicate: pred, To: kg.NodeID(to)}
}

func TestBulkIngestTriples(t *testing.T) {
	e := mkTestEngine(t)
	data := `# comment
alice	likes	bob
carol	trusts	dave
eve	hates	frank	-
`
	r := e.BulkIngestTriples(strings.NewReader(data), "bulk-test")
	if r.Total != 3 {
		t.Fatalf("total=%d, want 3", r.Total)
	}
	if r.Ingested != 3 {
		t.Fatalf("ingested=%d, want 3", r.Ingested)
	}
	if r.Errors != 0 {
		t.Fatalf("errors=%d", r.Errors)
	}
}

func TestBulkIngestCSV(t *testing.T) {
	e := mkTestEngine(t)
	data := "alice,likes,bob\ncarol,trusts,dave\n"
	r := e.BulkIngestCSV(strings.NewReader(data), "csv-test")
	if r.Ingested != 2 {
		t.Fatalf("ingested=%d, want 2", r.Ingested)
	}
}

func TestBulkIngestJSON(t *testing.T) {
	e := mkTestEngine(t)
	data := `[
		{"from":"alice","predicate":"likes","to":"bob"},
		{"from":"carol","predicate":"trusts","to":"dave","polarity":"-"}
	]`
	r := e.BulkIngestJSON(strings.NewReader(data), "json-test")
	if r.Ingested != 2 {
		t.Fatalf("ingested=%d, want 2", r.Ingested)
	}
	if r.Contradictions != 0 {
		t.Fatalf("contradictions=%d", r.Contradictions)
	}
}

func TestBulkAppliesRulesOnce(t *testing.T) {
	e := mkTestEngine(t)
	_ = e.AddRule(rules.Rule{
		Name: "likes-trans",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "likes", To: "?y"},
			{From: "?y", Predicate: "likes", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "likes", To: "?z"},
	})

	data := "alice\tlikes\tbob\nbob\tlikes\tcarol\n"
	r := e.BulkIngestTriples(strings.NewReader(data), "bulk-rule")
	if r.Ingested != 2 {
		t.Fatalf("ingested=%d", r.Ingested)
	}

	// Rule should have derived alice likes carol
	v, _ := e.Evaluate(mkAtom("alice", "likes", "carol"))
	if v != TruthTrue {
		t.Fatalf("derived=%v, want TruthTrue", v)
	}
}
