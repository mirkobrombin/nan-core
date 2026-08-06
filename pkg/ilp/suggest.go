package ilp

import (
	"sort"
	"strconv"

	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

// SuggestTransitive proposes transitive rules for predicates with enough observed two-hop chains.
func SuggestTransitive(edges []kg.Edge, minChains int) []Suggestion {
	if minChains <= 0 {
		minChains = 1
	}

	byPred := make(map[string][]kg.Edge, 8)
	for _, e := range edges {
		byPred[e.Predicate] = append(byPred[e.Predicate], e)
	}

	preds := make([]string, 0, len(byPred))
	for p := range byPred {
		preds = append(preds, p)
	}
	sort.Strings(preds)

	out := make([]Suggestion, 0, 8)
	for _, p := range preds {
		eds := byPred[p]

		// adjacency a->[]b for predicate p
		adj := make(map[string][]string, 16)
		for _, ed := range eds {
			adj[string(ed.From)] = append(adj[string(ed.From)], string(ed.To))
		}
		for k := range adj {
			sort.Strings(adj[k])
		}

		chains := 0
		seen := make(map[string]struct{}, 64)
		froms := make([]string, 0, len(adj))
		for f := range adj {
			froms = append(froms, f)
		}
		sort.Strings(froms)
		for _, a := range froms {
			for _, b := range adj[a] {
				for _, c := range adj[b] {
					k := a + "|" + b + "|" + c
					if _, ok := seen[k]; ok {
						continue
					}
					seen[k] = struct{}{}
					chains++
				}
			}
		}

		if chains < minChains {
			continue
		}

		r := rules.Rule{
			Name: "transitive:" + p,
			If: []rules.AtomPattern{
				{From: "?x", Predicate: p, To: "?y"},
				{From: "?y", Predicate: p, To: "?z"},
			},
			Then: rules.AtomPattern{From: "?x", Predicate: p, To: "?z"},
		}
		out = append(out, Suggestion{Rule: r, Reason: "observed_chains=" + strconv.Itoa(chains)})
	}
	return out
}
