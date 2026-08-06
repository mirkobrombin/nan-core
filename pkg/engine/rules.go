package engine

import (
	"errors"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
	"github.com/mirkobrombin/nan-core/pkg/store"
)

// Rules returns a copy of all active rules.
func (e *Engine) Rules() []rules.Rule {
	out := make([]rules.Rule, len(e.rules))
	copy(out, e.rules)
	return out
}

func (e *Engine) AddRule(r rules.Rule) error {
	if r.Name == "" {
		return errors.New("rule name must be non-empty")
	}
	if len(r.If) == 0 {
		return errors.New("rule must have premises")
	}

	e.rules = append(e.rules, r)

	if e.log != nil {
		payload, err := encodeRule(r)
		if err != nil {
			return err
		}
		if err := e.log.Append(store.Record{Type: store.EventRuleAdded, Payload: payload}); err != nil {
			return err
		}
	}

	e.ApplyRules(1000)
	return nil
}

func (e *Engine) ApplyRules(maxDerivations int) {
	if maxDerivations <= 0 {
		return
	}

	remaining := maxDerivations
	for remaining > 0 {
		edges := e.graph.EdgesAll()

		// Build predicate index once per round
		predIdx := rules.BuildPredicateIndex(edges)

		addedThisRound := 0
		for _, r := range e.rules {
			derivs := rules.EvaluateRuleDerivationsIndexed(r, edges, predIdx, remaining)
			for _, d := range derivs {
				atom := belief.Atom{From: kg.NodeID(d.Then.From), Predicate: d.Then.Predicate, To: kg.NodeID(d.Then.To)}

				prem := make([]belief.Atom, 0, len(d.Premises))
				for _, p := range d.Premises {
					prem = append(prem, belief.Atom{From: kg.NodeID(p.From), Predicate: p.Predicate, To: kg.NodeID(p.To)})
				}
				dp := DerivationProof{Rule: r.Name, Premises: prem}
				e.addDerivation(atom, dp)

				before := e.beliefs.Count()
				_, _ = e.ingestNoLog(belief.Belief{Atom: atom, Polarity: belief.PolarityPositive, Source: "rule:" + r.Name, UnixNano: 0})
				after := e.beliefs.Count()
				if after > before {
					addedThisRound++
					remaining--
					if remaining <= 0 {
						return
					}
				}
			}
		}
		if addedThisRound == 0 {
			return
		}
	}
}
