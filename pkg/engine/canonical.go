package engine

import (
	"sort"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

func (e *Engine) CanonicalBeliefs() []belief.Belief {
	atoms := e.beliefs.Atoms()
	out := make([]belief.Belief, 0, 64)
	for _, a := range atoms {
		out = append(out, e.beliefs.Snapshot(a)...)
	}
	sort.Slice(out, func(i, j int) bool {
		ki := atomKey(out[i].Atom)
		kj := atomKey(out[j].Atom)
		if ki != kj {
			return ki < kj
		}
		if out[i].Polarity != out[j].Polarity {
			return out[i].Polarity < out[j].Polarity
		}
		if out[i].UnixNano != out[j].UnixNano {
			return out[i].UnixNano < out[j].UnixNano
		}
		return out[i].Source < out[j].Source
	})
	return out
}

func (e *Engine) CanonicalResolutions() []belief.Resolution {
	out := e.beliefs.Resolutions()
	sort.Slice(out, func(i, j int) bool {
		ki := atomKey(out[i].Atom)
		kj := atomKey(out[j].Atom)
		if ki != kj {
			return ki < kj
		}
		if out[i].Polarity != out[j].Polarity {
			return out[i].Polarity < out[j].Polarity
		}
		if out[i].UnixNano != out[j].UnixNano {
			return out[i].UnixNano < out[j].UnixNano
		}
		return out[i].Source < out[j].Source
	})
	return out
}

func (e *Engine) RulesSnapshot() []rules.Rule {
	out := make([]rules.Rule, len(e.rules))
	copy(out, e.rules)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		// compare IF
		li := len(out[i].If)
		lj := len(out[j].If)
		lm := li
		if lj < lm {
			lm = lj
		}
		for k := 0; k < lm; k++ {
			ai := out[i].If[k]
			aj := out[j].If[k]
			if ai.From != aj.From {
				return ai.From < aj.From
			}
			if ai.Predicate != aj.Predicate {
				return ai.Predicate < aj.Predicate
			}
			if ai.To != aj.To {
				return ai.To < aj.To
			}
		}
		if li != lj {
			return li < lj
		}
		// THEN
		if out[i].Then.From != out[j].Then.From {
			return out[i].Then.From < out[j].Then.From
		}
		if out[i].Then.Predicate != out[j].Then.Predicate {
			return out[i].Then.Predicate < out[j].Then.Predicate
		}
		return out[i].Then.To < out[j].Then.To
	})
	return out
}
