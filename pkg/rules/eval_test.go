package rules

import (
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/kg"
)

func TestEvaluateRuleSimpleChain(t *testing.T) {
	edges := []kg.Edge{
		{From: "a", Predicate: "likes", To: "b"},
		{From: "b", Predicate: "likes", To: "c"},
	}

	r := Rule{
		Name: "transitive",
		If: []AtomPattern{
			{From: "?x", Predicate: "likes", To: "?y"},
			{From: "?y", Predicate: "likes", To: "?z"},
		},
		Then: AtomPattern{From: "?x", Predicate: "likes2", To: "?z"},
	}

	out := EvaluateRule(r, edges, 100)
	if len(out) != 1 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].From != "a" || out[0].Predicate != "likes2" || out[0].To != "c" {
		t.Fatalf("out=%+v", out[0])
	}
}
