package engine

import (
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
	"github.com/mirkobrombin/nan-core/pkg/store"
)

func atom(from, pred, to string) belief.Atom {
	return belief.Atom{From: kg.NodeID(from), Predicate: pred, To: kg.NodeID(to)}
}

func assertPositive(t *testing.T, e *Engine, from, pred, to string) {
	t.Helper()
	if _, err := e.Ingest(belief.Belief{Atom: atom(from, pred, to), Polarity: belief.PolarityPositive, Source: "test"}); err != nil {
		t.Fatalf("ingest %s %s %s: %v", from, pred, to, err)
	}
}

func transitiveRule() rules.Rule {
	return rules.Rule{
		Name: "isa-transitive",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "isa", To: "?y"},
			{From: "?y", Predicate: "isa", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "isa", To: "?z"},
	}
}

// A fact resolved false must not remain a premise for derivation, and the
// derivation that rested on it must not survive the resolution. This is the
// failure mode where a proof core keeps proving from a premise the store holds
// to be false.
func TestResolveFalseDropsDerivationFromFalsePremise(t *testing.T) {
	e, _ := New()
	assertPositive(t, e, "a", "isa", "b")
	assertPositive(t, e, "b", "isa", "c")
	if err := e.AddRule(transitiveRule()); err != nil {
		t.Fatal(err)
	}

	if v, _ := e.Evaluate(atom("a", "isa", "c")); v != TruthTrue {
		t.Fatalf("before resolution: a isa c = %d, want %d", v, TruthTrue)
	}

	if err := e.Resolve(belief.Resolution{Atom: atom("a", "isa", "b"), Polarity: belief.PolarityNegative, Source: "op"}); err != nil {
		t.Fatal(err)
	}

	if v, _ := e.Evaluate(atom("a", "isa", "b")); v != TruthFalse {
		t.Fatalf("a isa b after resolve-: %d, want %d", v, TruthFalse)
	}
	if v, _ := e.Evaluate(atom("a", "isa", "c")); v != TruthUnknown {
		t.Fatalf("a isa c after resolve-: %d, want %d (no true premise left)", v, TruthUnknown)
	}
	for _, ed := range e.Graph().EdgesFrom("a") {
		if ed.Predicate == "isa" && ed.To == "b" {
			t.Fatal("a isa b is still an edge in the graph after being resolved false")
		}
	}
}

// Resolving the premise back to true restores the derivation, so the fix is a
// projection of current truth, not a one-way delete.
func TestResolveTrueRestoresDerivation(t *testing.T) {
	e, _ := New()
	assertPositive(t, e, "a", "isa", "b")
	assertPositive(t, e, "b", "isa", "c")
	if err := e.AddRule(transitiveRule()); err != nil {
		t.Fatal(err)
	}

	if err := e.Resolve(belief.Resolution{Atom: atom("a", "isa", "b"), Polarity: belief.PolarityNegative, Source: "op"}); err != nil {
		t.Fatal(err)
	}
	if v, _ := e.Evaluate(atom("a", "isa", "c")); v != TruthUnknown {
		t.Fatalf("a isa c while premise false: %d, want %d", v, TruthUnknown)
	}

	if err := e.Resolve(belief.Resolution{Atom: atom("a", "isa", "b"), Polarity: belief.PolarityPositive, Source: "op"}); err != nil {
		t.Fatal(err)
	}
	if v, _ := e.Evaluate(atom("a", "isa", "c")); v != TruthTrue {
		t.Fatalf("a isa c after premise resolved true again: %d, want %d", v, TruthTrue)
	}
}

// Re-asserting a positive belief for an atom already resolved false must not
// sneak the premise back into the graph through the ingest path.
func TestIngestDoesNotReviveFalseResolvedPremise(t *testing.T) {
	e, _ := New()
	assertPositive(t, e, "a", "isa", "b")
	assertPositive(t, e, "b", "isa", "c")
	if err := e.AddRule(transitiveRule()); err != nil {
		t.Fatal(err)
	}
	if err := e.Resolve(belief.Resolution{Atom: atom("a", "isa", "b"), Polarity: belief.PolarityNegative, Source: "op"}); err != nil {
		t.Fatal(err)
	}

	assertPositive(t, e, "a", "isa", "b")

	if v, _ := e.Evaluate(atom("a", "isa", "c")); v != TruthUnknown {
		t.Fatalf("a isa c after re-asserting a false-resolved premise: %d, want %d", v, TruthUnknown)
	}
}

// The failure Pietro emphasised: a generation rebuilt from the log must apply
// resolutions before it constructs the graph, or replay reintroduces the
// derivation from a false premise the live engine had already dropped.
func TestReplayAppliesResolutionsBeforeGraph(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/nan.wal"

	w, err := store.OpenWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(WithWAL(w))
	if err != nil {
		t.Fatal(err)
	}
	assertPositive(t, e, "a", "isa", "b")
	assertPositive(t, e, "b", "isa", "c")
	if err := e.AddRule(transitiveRule()); err != nil {
		t.Fatal(err)
	}
	if err := e.Resolve(belief.Resolution{Atom: atom("a", "isa", "b"), Polarity: belief.PolarityNegative, Source: "op"}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	w2, err := store.OpenWAL(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w2.Close()
	replayed, err := New(WithWAL(w2))
	if err != nil {
		t.Fatal(err)
	}
	if err := replayed.ReplayFromLog(); err != nil {
		t.Fatal(err)
	}

	if v, _ := replayed.Evaluate(atom("a", "isa", "b")); v != TruthFalse {
		t.Fatalf("replayed a isa b: %d, want %d", v, TruthFalse)
	}
	if v, _ := replayed.Evaluate(atom("a", "isa", "c")); v != TruthUnknown {
		t.Fatalf("replayed a isa c: %d, want %d (replay reintroduced a false premise)", v, TruthUnknown)
	}
}
