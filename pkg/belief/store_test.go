package belief

import (
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/kg"
)

func TestContradictionDetected(t *testing.T) {
	s := NewStore()

	atom := Atom{From: kg.NodeID("u"), Predicate: "likes", To: kg.NodeID("coffee")}

	c, err := s.Add(Belief{Atom: atom, Polarity: PolarityPositive, Source: "s1", UnixNano: 1})
	if err != nil {
		t.Fatal(err)
	}
	if c != nil {
		t.Fatalf("unexpected contradiction on first insert")
	}

	c, err = s.Add(Belief{Atom: atom, Polarity: PolarityNegative, Source: "s2", UnixNano: 2})
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatalf("expected contradiction")
	}
	if c.Reason == "" {
		t.Fatalf("expected reason")
	}
}

func TestResolutionAuthoritative(t *testing.T) {
	s := NewStore()
	atom := Atom{From: kg.NodeID("a"), Predicate: "p", To: kg.NodeID("b")}

	if err := s.SetResolution(Resolution{Atom: atom, Polarity: PolarityPositive, Source: "op", UnixNano: 10}); err != nil {
		t.Fatal(err)
	}

	c, err := s.Add(Belief{Atom: atom, Polarity: PolarityNegative, Source: "s1", UnixNano: 11})
	if err != nil {
		t.Fatal(err)
	}
	if c == nil || c.Reason != "conflicts with resolution" {
		t.Fatalf("expected resolution contradiction, got=%+v", c)
	}
}

func TestDedup(t *testing.T) {
	s := NewStore()
	atom := Atom{From: kg.NodeID("a"), Predicate: "p", To: kg.NodeID("b")}

	_, _ = s.Add(Belief{Atom: atom, Polarity: PolarityPositive, Source: "s1", UnixNano: 1})
	_, _ = s.Add(Belief{Atom: atom, Polarity: PolarityPositive, Source: "s1", UnixNano: 1})

	if s.Count() != 1 {
		t.Fatalf("count=%d want 1", s.Count())
	}
}

func TestSnapshotDeterministicOrder(t *testing.T) {
	s := NewStore()
	atom := Atom{From: kg.NodeID("a"), Predicate: "p", To: kg.NodeID("b")}

	_, _ = s.Add(Belief{Atom: atom, Polarity: PolarityPositive, Source: "z", UnixNano: 2})
	_, _ = s.Add(Belief{Atom: atom, Polarity: PolarityPositive, Source: "a", UnixNano: 1})
	_, _ = s.Add(Belief{Atom: atom, Polarity: PolarityNegative, Source: "m", UnixNano: 3})

	snap := s.Snapshot(atom)
	if len(snap) != 3 {
		t.Fatalf("len=%d want 3", len(snap))
	}
	// negative first, then positive sorted by time then source
	if snap[0].Polarity != PolarityNegative {
		t.Fatalf("expected negative first")
	}
	if snap[1].UnixNano != 1 || snap[1].Source != "a" {
		t.Fatalf("unexpected snap[1]=%+v", snap[1])
	}
	if snap[2].UnixNano != 2 || snap[2].Source != "z" {
		t.Fatalf("unexpected snap[2]=%+v", snap[2])
	}
}
