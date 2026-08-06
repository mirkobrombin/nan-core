package engine

import (
	"path/filepath"
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/store"
)

func TestEngineIngestProjectsToKG(t *testing.T) {
	e, err := New()
	if err != nil {
		t.Fatal(err)
	}

	atom := belief.Atom{From: kg.NodeID("u"), Predicate: "likes", To: kg.NodeID("coffee")}
	_, err = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityPositive, Source: "s", UnixNano: 1})
	if err != nil {
		t.Fatal(err)
	}

	edges := e.Graph().EdgesFrom("u")
	if len(edges) != 1 {
		t.Fatalf("edges=%d want 1", len(edges))
	}
	if edges[0].Predicate != "likes" || edges[0].To != "coffee" {
		t.Fatalf("unexpected edge=%+v", edges[0])
	}
}

func TestEngineWALReplay(t *testing.T) {
	dir := t.TempDir()
	walPath := filepath.Join(dir, "nan.wal")

	w, err := store.OpenWAL(walPath)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	e, _ := New(WithLog(w))

	atom := belief.Atom{From: kg.NodeID("a"), Predicate: "p", To: kg.NodeID("b")}
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityPositive, Source: "s1", UnixNano: 1})
	_, _ = e.Ingest(belief.Belief{Atom: atom, Polarity: belief.PolarityNegative, Source: "s2", UnixNano: 2})

	e2, _ := New(WithLog(w))
	if err := e2.ReplayFromLog(); err != nil {
		t.Fatal(err)
	}

	if e2.BeliefStore().Count() != 2 {
		t.Fatalf("count=%d want 2", e2.BeliefStore().Count())
	}

	edges := e2.Graph().EdgesFrom("a")
	if len(edges) != 1 {
		t.Fatalf("edges=%d want 1", len(edges))
	}
}
