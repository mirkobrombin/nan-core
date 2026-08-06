package engine

import (
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

func TestEvaluateDerivedProof(t *testing.T) {
	e, _ := New()

	_, _ = e.Ingest(belief.Belief{Atom: belief.Atom{From: kg.NodeID("a"), Predicate: "likes", To: kg.NodeID("b")}, Polarity: belief.PolarityPositive, Source: "s", UnixNano: 1})
	_, _ = e.Ingest(belief.Belief{Atom: belief.Atom{From: kg.NodeID("b"), Predicate: "likes", To: kg.NodeID("c")}, Polarity: belief.PolarityPositive, Source: "s", UnixNano: 2})

	_ = e.AddRule(rules.Rule{
		Name: "transitive",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "likes", To: "?y"},
			{From: "?y", Predicate: "likes", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "likes2", To: "?z"},
	})

	atom := belief.Atom{From: kg.NodeID("a"), Predicate: "likes2", To: kg.NodeID("c")}
	v, pf := e.Evaluate(atom)
	if v != TruthTrue {
		t.Fatalf("v=%v", v)
	}
	if pf.Kind != "derived" {
		t.Fatalf("kind=%s", pf.Kind)
	}
	if len(pf.Derivations) == 0 {
		t.Fatalf("missing derivations")
	}
	if pf.Derivations[0].Rule != "transitive" {
		t.Fatalf("rule=%s", pf.Derivations[0].Rule)
	}
	if len(pf.Derivations[0].Premises) != 2 {
		t.Fatalf("prem=%+v", pf.Derivations[0].Premises)
	}
}
