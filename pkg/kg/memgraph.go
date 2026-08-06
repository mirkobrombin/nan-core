package kg

import (
	"errors"
	"sort"
	"sync"
)

type MemGraph struct {
	mu sync.RWMutex

	// out[from][predicate][to] = struct{}
	out map[NodeID]map[string]map[NodeID]struct{}
	// in[to][predicate][from] = struct{}
	in map[NodeID]map[string]map[NodeID]struct{}

	facts int
}

func NewMemGraph() *MemGraph {
	return &MemGraph{
		out: make(map[NodeID]map[string]map[NodeID]struct{}),
		in:  make(map[NodeID]map[string]map[NodeID]struct{}),
	}
}

func (g *MemGraph) AddFact(from NodeID, predicate string, to NodeID) error {
	if from == "" || to == "" {
		return errors.New("from/to must be non-empty")
	}
	if predicate == "" {
		return errors.New("predicate must be non-empty")
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if _, ok := g.out[from]; !ok {
		g.out[from] = make(map[string]map[NodeID]struct{})
	}
	if _, ok := g.out[from][predicate]; !ok {
		g.out[from][predicate] = make(map[NodeID]struct{})
	}

	if _, ok := g.in[to]; !ok {
		g.in[to] = make(map[string]map[NodeID]struct{})
	}
	if _, ok := g.in[to][predicate]; !ok {
		g.in[to][predicate] = make(map[NodeID]struct{})
	}

	if _, exists := g.out[from][predicate][to]; exists {
		return nil
	}
	g.out[from][predicate][to] = struct{}{}
	g.in[to][predicate][from] = struct{}{}
	g.facts++
	return nil
}

func (g *MemGraph) EdgesFrom(from NodeID) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	preds, ok := g.out[from]
	if !ok {
		return nil
	}

	edges := make([]Edge, 0, 8)
	for pred, tos := range preds {
		for to := range tos {
			edges = append(edges, Edge{From: from, Predicate: pred, To: to})
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Predicate != edges[j].Predicate {
			return edges[i].Predicate < edges[j].Predicate
		}
		return edges[i].To < edges[j].To
	})
	return edges
}

func (g *MemGraph) EdgesTo(to NodeID) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	preds, ok := g.in[to]
	if !ok {
		return nil
	}

	edges := make([]Edge, 0, 8)
	for pred, froms := range preds {
		for from := range froms {
			edges = append(edges, Edge{From: from, Predicate: pred, To: to})
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Predicate != edges[j].Predicate {
			return edges[i].Predicate < edges[j].Predicate
		}
		return edges[i].From < edges[j].From
	})
	return edges
}

func (g *MemGraph) EdgesAll() []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edges := make([]Edge, 0, g.facts)
	for from, preds := range g.out {
		for pred, tos := range preds {
			for to := range tos {
				edges = append(edges, Edge{From: from, Predicate: pred, To: to})
			}
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].Predicate != edges[j].Predicate {
			return edges[i].Predicate < edges[j].Predicate
		}
		return edges[i].To < edges[j].To
	})
	return edges
}

func (g *MemGraph) FactsCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.facts
}

// CountIncoming returns how many distinct subjects satisfy (?x predicate to).
func (g *MemGraph) CountIncoming(to NodeID, predicate string) int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	preds, ok := g.in[to]
	if !ok {
		return 0
	}
	froms, ok := preds[predicate]
	if !ok {
		return 0
	}
	return len(froms)
}

// EdgesByPredicate returns all edges with a given predicate in deterministic order.
func (g *MemGraph) EdgesByPredicate(predicate string) []Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edges := make([]Edge, 0, 16)
	for from, preds := range g.out {
		tos, ok := preds[predicate]
		if !ok {
			continue
		}
		for to := range tos {
			edges = append(edges, Edge{From: from, Predicate: predicate, To: to})
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})
	return edges
}

// Predicates returns all unique predicates in the graph in sorted order.
func (g *MemGraph) Predicates() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	set := make(map[string]struct{}, 32)
	for _, preds := range g.out {
		for p := range preds {
			set[p] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
