package engine

import (
	"fmt"
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

// Symbolic Turing Test: the engine must pass these deterministic challenges
// that verify it never hallucinates, always justifies, and handles edge cases.

func TestTuringNeverInventsFacts(t *testing.T) {
	e, _ := New()
	// Query something never asserted
	v, pf := e.Evaluate(belief.Atom{From: "x", Predicate: "likes", To: "y"})
	if v != TruthUnknown {
		t.Fatalf("invented a fact: truth=%d", v)
	}
	if pf.Kind != "none" {
		t.Fatalf("proof should be none, got %s", pf.Kind)
	}
}

func TestTuringContradictionBlocksAnswer(t *testing.T) {
	e, _ := New()
	atom := belief.Atom{From: "a", Predicate: "is", To: "b"}
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityPositive, Source: "t"})
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityNegative, Source: "t"})

	v, pf := e.Evaluate(atom)
	if v != TruthBoth {
		t.Fatalf("should report contradiction, got truth=%d", v)
	}
	if pf.Kind != "contradiction" {
		t.Fatalf("proof should be contradiction, got %s", pf.Kind)
	}
}

func TestTuringResolutionOverridesContradiction(t *testing.T) {
	e, _ := New()
	atom := belief.Atom{From: "a", Predicate: "is", To: "b"}
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityPositive, Source: "t"})
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityNegative, Source: "t"})

	_ = e.Resolve(belief.Resolution{Atom: atom, Polarity: belief.PolarityPositive, Source: "authority"})

	v, pf := e.Evaluate(atom)
	if v != TruthTrue {
		t.Fatalf("resolution should override, got truth=%d", v)
	}
	if pf.Kind != "resolution" {
		t.Fatalf("proof should be resolution, got %s", pf.Kind)
	}
}

func TestTuringDerivedFactHasJustification(t *testing.T) {
	e, _ := New()
	_ = e.AddRule(rules.Rule{
		Name: "friend-of-friend",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "friend", To: "?y"},
			{From: "?y", Predicate: "friend", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "friend", To: "?z"},
	})
	_, _ = e.Ingest(belief.Belief{
		Atom: belief.Atom{From: "alice", Predicate: "friend", To: "bob"}, Polarity: belief.PolarityPositive, Source: "t",
	})
	_, _ = e.Ingest(belief.Belief{
		Atom: belief.Atom{From: "bob", Predicate: "friend", To: "carol"}, Polarity: belief.PolarityPositive, Source: "t",
	})

	v, pf := e.Evaluate(belief.Atom{From: "alice", Predicate: "friend", To: "carol"})
	if v != TruthTrue {
		t.Fatalf("derived fact should be true, got %d", v)
	}
	if pf.Kind != "derived" {
		t.Fatalf("proof should be derived, got %s", pf.Kind)
	}
	if len(pf.Derivations) == 0 {
		t.Fatal("derivation chain must be non-empty")
	}
}

func TestTuringNoSelfAmplification(t *testing.T) {
	// Ensure rules don't create infinite self-referencing facts
	e, _ := New()
	_ = e.AddRule(rules.Rule{
		Name: "self",
		If:   []rules.AtomPattern{{From: "?x", Predicate: "is", To: "?y"}},
		Then: rules.AtomPattern{From: "?y", Predicate: "is", To: "?x"},
	})
	_, _ = e.Ingest(belief.Belief{
		Atom: belief.Atom{From: "a", Predicate: "is", To: "b"}, Polarity: belief.PolarityPositive, Source: "t",
	})

	// Should terminate (maxDerivations cap) and not infinite-loop
	v, _ := e.Evaluate(belief.Atom{From: "b", Predicate: "is", To: "a"})
	if v != TruthTrue {
		t.Fatalf("symmetric rule should derive b is a, got %d", v)
	}
}

func TestTuringProofGraphDeterministic(t *testing.T) {
	e, _ := New()
	_ = e.AddRule(rules.Rule{
		Name: "chain",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "r", To: "?y"},
			{From: "?y", Predicate: "r", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "r", To: "?z"},
	})
	for i := 0; i < 5; i++ {
		_, _ = e.Ingest(belief.Belief{
			Atom:     belief.Atom{From: kg.NodeID(fmt.Sprintf("n%d", i)), Predicate: "r", To: kg.NodeID(fmt.Sprintf("n%d", i+1))},
			Polarity: belief.PolarityPositive, Source: "t",
		})
	}

	target := belief.Atom{From: "n0", Predicate: "r", To: "n3"}
	g1 := e.ExplainGraph(target, 3)
	g2 := e.ExplainGraph(target, 3)

	b1, _ := g1.MarshalJSONPretty()
	b2, _ := g2.MarshalJSONPretty()
	if string(b1) != string(b2) {
		t.Fatalf("proof graph is non-deterministic")
	}
}

func TestTuringMetricsAccurate(t *testing.T) {
	e, _ := New()
	atom := belief.Atom{From: "x", Predicate: "y", To: "z"}
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityPositive, Source: "t"})
	e.Evaluate(atom)
	e.Evaluate(atom)

	snap := e.Metrics().Snapshot()
	if snap.Ingest != 1 {
		t.Fatalf("ingest=%d, want 1", snap.Ingest)
	}
	if snap.Query != 2 {
		t.Fatalf("query=%d, want 2", snap.Query)
	}
}
