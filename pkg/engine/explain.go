package engine

import (
	"encoding/json"
	"sort"

	"github.com/mirkobrombin/nan-core/pkg/belief"
)

type ProofNode struct {
	ID      string            `json:"id"`
	Kind    string            `json:"kind"`
	Atom    belief.Atom       `json:"atom"`
	Details map[string]string `json:"details,omitempty"`
}

type ProofEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
	Rule string `json:"rule,omitempty"`
}

type ProofGraph struct {
	Root  string      `json:"root"`
	Nodes []ProofNode `json:"nodes"`
	Edges []ProofEdge `json:"edges"`
}

func (g ProofGraph) MarshalJSONPretty() ([]byte, error) {
	return json.MarshalIndent(g, "", "  ")
}

func (e *Engine) ExplainGraph(root belief.Atom, depth int) ProofGraph {
	if depth < 0 {
		depth = 0
	}

	nodes := make(map[string]ProofNode, 32)
	edges := make([]ProofEdge, 0, 32)
	seen := make(map[string]int, 32)

	var visit func(a belief.Atom, d int)
	visit = func(a belief.Atom, d int) {
		id := atomKey(a)
		if prev, ok := seen[id]; ok && prev >= d {
			return
		}
		seen[id] = d

		if _, ok := nodes[id]; !ok {
			v, pf := e.Evaluate(a)
			kind := pf.Kind
			if v == TruthBoth {
				kind = "contradiction"
			}
			nodes[id] = ProofNode{ID: id, Kind: kind, Atom: a, Details: pf.Details}
		}

		if d <= 0 {
			return
		}

		for _, der := range e.derivationsFor(a) {
			for _, p := range der.Premises {
				pid := atomKey(p)
				edges = append(edges, ProofEdge{From: id, To: pid, Kind: "premise", Rule: der.Rule})
				visit(p, d-1)
			}
		}
	}

	visit(root, depth)

	nlist := make([]ProofNode, 0, len(nodes))
	for _, n := range nodes {
		nlist = append(nlist, n)
	}
	sort.Slice(nlist, func(i, j int) bool { return nlist[i].ID < nlist[j].ID })

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		if edges[i].Kind != edges[j].Kind {
			return edges[i].Kind < edges[j].Kind
		}
		return edges[i].Rule < edges[j].Rule
	})

	return ProofGraph{Root: atomKey(root), Nodes: nlist, Edges: edges}
}
