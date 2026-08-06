package engine

import (
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
)

func TestEvaluateResolutionWins(t *testing.T) {
	e, _ := New()
	atom := belief.Atom{From: kg.NodeID("a"), Predicate: "p", To: kg.NodeID("b")}

	_ = e.Resolve(belief.Resolution{Atom: atom, Polarity: belief.PolarityNegative, Source: "op", UnixNano: 1})
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityPositive, Source: "s", UnixNano: 2})

	v, pf := e.Evaluate(atom)
	if v != TruthFalse {
		t.Fatalf("v=%v want %v", v, TruthFalse)
	}
	if pf.Kind != "resolution" {
		t.Fatalf("kind=%s", pf.Kind)
	}
}

func TestEvaluateBoth(t *testing.T) {
	e, _ := New()
	atom := belief.Atom{From: kg.NodeID("a"), Predicate: "p", To: kg.NodeID("b")}

	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityPositive, Source: "s1", UnixNano: 1})
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityNegative, Source: "s2", UnixNano: 2})

	v, _ := e.Evaluate(atom)
	if v != TruthBoth {
		t.Fatalf("v=%v want %v", v, TruthBoth)
	}
}
