package rules

import (
	"sort"
	"strings"

	"github.com/mirkobrombin/nan-core/pkg/kg"
)

// PredicateIndex maps predicate strings to sorted EdgeViews.
type PredicateIndex map[string][]EdgeView

// BuildPredicateIndex creates a predicate→edges lookup from a raw edge set.
// Each bucket is sorted deterministically.
func BuildPredicateIndex(edges []kg.Edge) PredicateIndex {
	idx := make(PredicateIndex, 32)
	for _, e := range edges {
		ev := EdgeView{From: string(e.From), Predicate: e.Predicate, To: string(e.To)}
		idx[e.Predicate] = append(idx[e.Predicate], ev)
	}
	for _, bucket := range idx {
		sort.Slice(bucket, func(i, j int) bool {
			if bucket[i].From != bucket[j].From {
				return bucket[i].From < bucket[j].From
			}
			return bucket[i].To < bucket[j].To
		})
	}
	return idx
}

// EvaluateRuleDerivationsIndexed is like EvaluateRuleDerivations but uses a
// predicate index to skip irrelevant edges when the premise has a concrete
// (non-variable) predicate. Falls back to full scan for variable predicates.
func EvaluateRuleDerivationsIndexed(rule Rule, edges []kg.Edge, idx PredicateIndex, max int) []Derivation {
	if max <= 0 {
		return nil
	}

	allViews := edgesToView(edges)
	out := make([]Derivation, 0, 8)
	seen := make(map[string]struct{}, 16)

	key := func(d Derivation) string {
		b := strings.Builder{}
		b.WriteString(d.Then.From)
		b.WriteByte('|')
		b.WriteString(d.Then.Predicate)
		b.WriteByte('|')
		b.WriteString(d.Then.To)
		for _, p := range d.Premises {
			b.WriteByte(';')
			b.WriteString(p.From)
			b.WriteByte('|')
			b.WriteString(p.Predicate)
			b.WriteByte('|')
			b.WriteString(p.To)
		}
		return b.String()
	}

	// Resolve the candidate edges for a premise: use index if predicate is constant
	candidatesFor := func(prem AtomPattern, binds Bindings) []EdgeView {
		pred := prem.Predicate
		// If predicate is a variable, check bindings first
		if strings.HasPrefix(pred, "?") {
			if bound, ok := binds[pred]; ok {
				pred = bound
			} else {
				return allViews // variable not yet bound → full scan
			}
		}
		if bucket, ok := idx[pred]; ok {
			return bucket
		}
		return nil // no edges with this predicate
	}

	var dfs func(i int, binds Bindings)
	dfs = func(i int, binds Bindings) {
		if len(out) >= max {
			return
		}
		if i == len(rule.If) {
			then, ok := binds.Apply(rule.Then)
			if !ok {
				return
			}
			prem := make([]AtomPattern, 0, len(rule.If))
			for _, p := range rule.If {
				gp, ok := binds.Apply(p)
				if !ok {
					return
				}
				prem = append(prem, gp)
			}
			d := Derivation{Then: then, Premises: prem}
			k := key(d)
			if _, ok := seen[k]; ok {
				return
			}
			seen[k] = struct{}{}
			out = append(out, d)
			return
		}

		prem := rule.If[i]
		cands := candidatesFor(prem, binds)
		for _, ev := range cands {
			b2, ok := matchAtom(prem, ev, binds)
			if !ok {
				continue
			}
			dfs(i+1, b2)
			if len(out) >= max {
				return
			}
		}
	}

	dfs(0, Bindings{})

	sort.Slice(out, func(i, j int) bool {
		ai := out[i]
		aj := out[j]
		if ai.Then.From != aj.Then.From {
			return ai.Then.From < aj.Then.From
		}
		if ai.Then.Predicate != aj.Then.Predicate {
			return ai.Then.Predicate < aj.Then.Predicate
		}
		if ai.Then.To != aj.Then.To {
			return ai.Then.To < aj.Then.To
		}
		for k := 0; k < len(ai.Premises) && k < len(aj.Premises); k++ {
			pi := ai.Premises[k]
			pj := aj.Premises[k]
			if pi.From != pj.From {
				return pi.From < pj.From
			}
			if pi.Predicate != pj.Predicate {
				return pi.Predicate < pj.Predicate
			}
			if pi.To != pj.To {
				return pi.To < pj.To
			}
		}
		return len(ai.Premises) < len(aj.Premises)
	})
	return out
}
