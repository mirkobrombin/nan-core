package ilp

import (
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/kg"
)

func TestSuggestTransitiveFindsChains(t *testing.T) {
	edges := []kg.Edge{
		{From: "a", Predicate: "knows", To: "b"},
		{From: "b", Predicate: "knows", To: "c"},
		{From: "d", Predicate: "knows", To: "e"},
		{From: "e", Predicate: "knows", To: "f"},
		{From: "x", Predicate: "likes", To: "y"},
	}

	suggestions := SuggestTransitive(edges, 2)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion for 'knows', got %d", len(suggestions))
	}
	if suggestions[0].Rule.Name != "transitive:knows" {
		t.Fatalf("expected rule name transitive:knows, got %s", suggestions[0].Rule.Name)
	}
	if suggestions[0].Reason == "" {
		t.Fatal("expected non-empty reason")
	}
}

func TestSuggestTransitiveNoChainsBelow(t *testing.T) {
	edges := []kg.Edge{
		{From: "a", Predicate: "likes", To: "b"},
		{From: "c", Predicate: "likes", To: "d"},
	}

	suggestions := SuggestTransitive(edges, 2)
	if len(suggestions) != 0 {
		t.Fatalf("expected 0 suggestions, got %d", len(suggestions))
	}
}

func TestSuggestTransitiveSingleChain(t *testing.T) {
	edges := []kg.Edge{
		{From: "a", Predicate: "trusts", To: "b"},
		{From: "b", Predicate: "trusts", To: "c"},
	}

	suggestions := SuggestTransitive(edges, 1)
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
}

func TestNoopInducer(t *testing.T) {
	ind := NoopInducer{}
	result := ind.Observe(Example{})
	if len(result) != 0 {
		t.Fatalf("noop should produce no suggestions, got %d", len(result))
	}
}
