package engine

import (
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

func TestRuleDerivationAddsFactToKG(t *testing.T) {
	e, _ := New()

	atom1 := belief.Atom{From: kg.NodeID("a"), Predicate: "likes", To: kg.NodeID("b")}
	atom2 := belief.Atom{From: kg.NodeID("b"), Predicate: "likes", To: kg.NodeID("c")}
	_, _ = e.Ingest(belief.Belief{Atom: atom1, Polarity: belief.PolarityPositive, Source: "s", UnixNano: 1})
	_, _ = e.Ingest(belief.Belief{Atom: atom2, Polarity: belief.PolarityPositive, Source: "s", UnixNano: 2})

	r := rules.Rule{
		Name: "transitive",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "likes", To: "?y"},
			{From: "?y", Predicate: "likes", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "likes2", To: "?z"},
	}
	if err := e.AddRule(r); err != nil {
		t.Fatal(err)
	}

	edges := e.Graph().EdgesFrom("a")
	found := false
	for _, ed := range edges {
		if ed.Predicate == "likes2" && ed.To == "c" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected derived edge")
	}
}
