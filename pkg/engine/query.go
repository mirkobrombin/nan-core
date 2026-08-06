package engine

import (
	"time"

	"github.com/mirkobrombin/nan-core/pkg/belief"
)

type TruthValue uint8

const (
	TruthUnknown TruthValue = 0
	TruthTrue    TruthValue = 1
	TruthFalse   TruthValue = 2
	TruthBoth    TruthValue = 3
)

type Proof struct {
	Kind        string
	Details     map[string]string
	Derivations []DerivationProof
}

func (e *Engine) Evaluate(atom belief.Atom) (TruthValue, Proof) {
	t0 := time.Now()
	defer func() {
		e.metrics.QueryCount.Add(1)
		e.metrics.Record("", "evaluate", time.Since(t0), nil)
	}()

	if r, ok := e.beliefs.Resolution(atom); ok {
		if r.Polarity == belief.PolarityPositive {
			return TruthTrue, Proof{Kind: "resolution", Details: map[string]string{"source": r.Source}}
		}
		return TruthFalse, Proof{Kind: "resolution", Details: map[string]string{"source": r.Source}}
	}

	snap := e.beliefs.Snapshot(atom)
	seenPos := false
	seenNeg := false
	for _, b := range snap {
		if b.Polarity == belief.PolarityPositive {
			seenPos = true
		}
		if b.Polarity == belief.PolarityNegative {
			seenNeg = true
		}
	}

	switch {
	case seenPos && seenNeg:
		return TruthBoth, Proof{Kind: "contradiction", Details: map[string]string{"reason": "both polarities present"}}
	case seenPos:
		ds := e.derivationsFor(atom)
		if len(ds) > 0 {
			return TruthTrue, Proof{Kind: "derived", Details: map[string]string{"reason": "rule-derived"}, Derivations: ds}
		}
		return TruthTrue, Proof{Kind: "belief", Details: map[string]string{"reason": "positive belief present"}}
	case seenNeg:
		return TruthFalse, Proof{Kind: "belief", Details: map[string]string{"reason": "negative belief present"}}
	default:
		return TruthUnknown, Proof{Kind: "none", Details: map[string]string{"reason": "no evidence"}}
	}
}
