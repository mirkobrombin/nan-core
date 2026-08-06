package rules

import (
	"sort"
	"strings"

	"github.com/mirkobrombin/nan-core/pkg/kg"
)

type EdgeView struct {
	From      string
	Predicate string
	To        string
}

func edgesToView(edges []kg.Edge) []EdgeView {
	out := make([]EdgeView, 0, len(edges))
	for _, e := range edges {
		out = append(out, EdgeView{From: string(e.From), Predicate: e.Predicate, To: string(e.To)})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].Predicate != out[j].Predicate {
			return out[i].Predicate < out[j].Predicate
		}
		return out[i].To < out[j].To
	})
	return out
}

func matchAtom(p AtomPattern, e EdgeView, binds Bindings) (Bindings, bool) {
	b1, ok := unifyField(p.From, e.From, binds)
	if !ok {
		return nil, false
	}
	b2, ok := unifyField(p.Predicate, e.Predicate, b1)
	if !ok {
		return nil, false
	}
	b3, ok := unifyField(p.To, e.To, b2)
	if !ok {
		return nil, false
	}
	return b3, true
}

// EvaluateRule returns derived atoms (fully grounded) in deterministic order.
func EvaluateRule(rule Rule, edges []kg.Edge, max int) []AtomPattern {
	if max <= 0 {
		return nil
	}

	views := edgesToView(edges)
	results := make([]AtomPattern, 0, 8)

	var dfs func(i int, binds Bindings)
	dfs = func(i int, binds Bindings) {
		if len(results) >= max {
			return
		}
		if i == len(rule.If) {
			then, ok := binds.Apply(rule.Then)
			if ok {
				results = append(results, then)
			}
			return
		}

		prem := rule.If[i]
		for _, ev := range views {
			b2, ok := matchAtom(prem, ev, binds)
			if !ok {
				continue
			}
			dfs(i+1, b2)
			if len(results) >= max {
				return
			}
		}
	}

	dfs(0, Bindings{})

	sort.Slice(results, func(i, j int) bool {
		if results[i].From != results[j].From {
			return results[i].From < results[j].From
		}
		if results[i].Predicate != results[j].Predicate {
			return results[i].Predicate < results[j].Predicate
		}
		return results[i].To < results[j].To
	})
	return results
}

type Derivation struct {
	Then     AtomPattern
	Premises []AtomPattern
}

// EvaluateRuleDerivations returns derived atoms with their grounded premises in deterministic order.
func EvaluateRuleDerivations(rule Rule, edges []kg.Edge, max int) []Derivation {
	if max <= 0 {
		return nil
	}

	views := edgesToView(edges)
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
		for _, ev := range views {
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
		// tie-break on premises (in rule order)
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
