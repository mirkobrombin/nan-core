package engine

import "github.com/mirkobrombin/nan-core/pkg/belief"

func (e *Engine) addDerivation(atom belief.Atom, d DerivationProof) {
	e.derivMu.Lock()
	defer e.derivMu.Unlock()
	cur := e.derivations[atom]
	k := derivationKey(d)
	for _, ex := range cur {
		if derivationKey(ex) == k {
			return
		}
	}
	cur = append(cur, d)
	sortDerivations(cur)
	e.derivations[atom] = cur
}

func (e *Engine) derivationsFor(atom belief.Atom) []DerivationProof {
	e.derivMu.RLock()
	defer e.derivMu.RUnlock()
	cur := e.derivations[atom]
	out := make([]DerivationProof, len(cur))
	copy(out, cur)
	return out
}
