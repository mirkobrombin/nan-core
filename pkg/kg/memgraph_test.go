package kg

import (
	"sync"
	"testing"
)

func TestMemGraphAddAndQueryDeterministicOrder(t *testing.T) {
	g := NewMemGraph()
	_ = g.AddFact("a", "p2", "c")
	_ = g.AddFact("a", "p1", "b")
	_ = g.AddFact("a", "p1", "a")

	edges := g.EdgesFrom("a")
	if len(edges) != 3 {
		t.Fatalf("edges=%d want 3", len(edges))
	}

	if edges[0].Predicate != "p1" || edges[0].To != "a" {
		t.Fatalf("unexpected edges[0]=%+v", edges[0])
	}
	if edges[1].Predicate != "p1" || edges[1].To != "b" {
		t.Fatalf("unexpected edges[1]=%+v", edges[1])
	}
	if edges[2].Predicate != "p2" || edges[2].To != "c" {
		t.Fatalf("unexpected edges[2]=%+v", edges[2])
	}
}

func TestMemGraphConcurrentWritesDedup(t *testing.T) {
	g := NewMemGraph()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = g.AddFact("a", "p", "b")
		}()
	}
	wg.Wait()

	if g.FactsCount() != 1 {
		t.Fatalf("facts=%d want 1", g.FactsCount())
	}
}
