package engine

import (
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

// End-to-end integration test: full pipeline from ingest through rules,
// derivation, proof, semantic query, and contradiction resolution.
func TestE2EFullPipeline(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}

	// Step 1: ingest base facts
	facts := []belief.Belief{
		{Atom: belief.Atom{From: "alice", Predicate: "trusts", To: "bob"}, Polarity: belief.PolarityPositive, Source: "user"},
		{Atom: belief.Atom{From: "bob", Predicate: "trusts", To: "carol"}, Polarity: belief.PolarityPositive, Source: "user"},
		{Atom: belief.Atom{From: "carol", Predicate: "trusts", To: "dave"}, Polarity: belief.PolarityPositive, Source: "user"},
		{Atom: belief.Atom{From: "alice", Predicate: "likes", To: "carol"}, Polarity: belief.PolarityPositive, Source: "user"},
	}
	for _, f := range facts {
		if _, err := e.Ingest(f); err != nil {
			t.Fatalf("ingest: %v", err)
		}
	}

	// Step 2: verify direct fact
	v, pf := e.Evaluate(belief.Atom{From: "alice", Predicate: "trusts", To: "bob"})
	if v != TruthTrue || pf.Kind != "belief" {
		t.Fatalf("direct fact: truth=%d, proof=%s", v, pf.Kind)
	}

	// Step 3: add transitive rule
	err = e.AddRule(rules.Rule{
		Name: "trust-transitive",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "trusts", To: "?y"},
			{From: "?y", Predicate: "trusts", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "trusts", To: "?z"},
	})
	if err != nil {
		t.Fatalf("add rule: %v", err)
	}

	// Step 4: verify derived fact
	v, pf = e.Evaluate(belief.Atom{From: "alice", Predicate: "trusts", To: "carol"})
	if v != TruthTrue {
		t.Fatalf("derived fact should be true, got %d", v)
	}
	if pf.Kind != "derived" {
		t.Fatalf("proof should be derived, got %s", pf.Kind)
	}
	if len(pf.Derivations) == 0 {
		t.Fatal("derivation chain empty")
	}

	// Step 5: verify multi-hop derivation (alice trusts dave via bob→carol→dave)
	v, pf = e.Evaluate(belief.Atom{From: "alice", Predicate: "trusts", To: "dave"})
	if v != TruthTrue {
		t.Fatalf("multi-hop derived fact should be true, got %d", v)
	}

	// Step 6: proof graph is non-empty and deterministic
	g1 := e.ExplainGraph(belief.Atom{From: "alice", Predicate: "trusts", To: "dave"}, 4)
	if len(g1.Nodes) < 2 {
		t.Fatalf("proof graph too small: %d nodes", len(g1.Nodes))
	}
	g2 := e.ExplainGraph(belief.Atom{From: "alice", Predicate: "trusts", To: "dave"}, 4)
	b1, _ := g1.MarshalJSONPretty()
	b2, _ := g2.MarshalJSONPretty()
	if string(b1) != string(b2) {
		t.Fatal("proof graph non-deterministic")
	}

	// Step 7: inject contradiction
	_, err = e.Ingest(belief.Belief{
		Atom:     belief.Atom{From: "alice", Predicate: "likes", To: "carol"},
		Polarity: belief.PolarityNegative,
		Source:   "user2",
	})
	if err != nil {
		t.Fatalf("contradiction ingest: %v", err)
	}
	v, _ = e.Evaluate(belief.Atom{From: "alice", Predicate: "likes", To: "carol"})
	if v != TruthBoth {
		t.Fatalf("contradiction should give TruthBoth, got %d", v)
	}

	// Step 8: resolve contradiction
	err = e.Resolve(belief.Resolution{
		Atom:     belief.Atom{From: "alice", Predicate: "likes", To: "carol"},
		Polarity: belief.PolarityPositive,
		Source:   "authority",
	})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	v, pf = e.Evaluate(belief.Atom{From: "alice", Predicate: "likes", To: "carol"})
	if v != TruthTrue {
		t.Fatalf("resolved should be true, got %d", v)
	}
	if pf.Kind != "resolution" {
		t.Fatalf("proof should be resolution, got %s", pf.Kind)
	}

	// Step 9: query unknown fact — no hallucination
	v, pf = e.Evaluate(belief.Atom{From: "alice", Predicate: "hates", To: "bob"})
	if v != TruthUnknown {
		t.Fatalf("unknown fact should be Unknown, got %d", v)
	}

	// Step 10: semantic query
	results := e.SemanticQuery(
		belief.Atom{From: "alice", Predicate: "trusts", To: "bob"},
		0.5,
	)
	if len(results) == 0 {
		t.Fatal("semantic query returned no results")
	}
	// The exact match should be in results
	found := false
	for _, r := range results {
		if r.Atom.From == "alice" && r.Atom.Predicate == "trusts" && r.Atom.To == kg.NodeID("bob") && r.Similarity == 1.0 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("semantic query did not find exact match")
	}

	// Step 11: verify metrics
	snap := e.Metrics().Snapshot()
	if snap.Ingest < 5 {
		t.Fatalf("ingest count too low: %d", snap.Ingest)
	}
	if snap.Query < 6 {
		t.Fatalf("query count too low: %d", snap.Query)
	}
	if snap.Resolve < 1 {
		t.Fatalf("resolve count too low: %d", snap.Resolve)
	}
}
