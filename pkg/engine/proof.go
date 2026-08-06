package engine

import (
	"sort"
	"strings"

	"github.com/mirkobrombin/nan-core/pkg/belief"
)

type DerivationProof struct {
	Rule     string
	Premises []belief.Atom
}

func atomKey(a belief.Atom) string {
	b := strings.Builder{}
	b.WriteString(string(a.From))
	b.WriteByte('|')
	b.WriteString(a.Predicate)
	b.WriteByte('|')
	b.WriteString(string(a.To))
	return b.String()
}

func derivationKey(d DerivationProof) string {
	b := strings.Builder{}
	b.WriteString(d.Rule)
	for _, p := range d.Premises {
		b.WriteByte(';')
		b.WriteString(atomKey(p))
	}
	return b.String()
}

func sortDerivations(ds []DerivationProof) {
	sort.Slice(ds, func(i, j int) bool {
		if ds[i].Rule != ds[j].Rule {
			return ds[i].Rule < ds[j].Rule
		}
		pi := ds[i].Premises
		pj := ds[j].Premises
		for k := 0; k < len(pi) && k < len(pj); k++ {
			ki := atomKey(pi[k])
			kj := atomKey(pj[k])
			if ki != kj {
				return ki < kj
			}
		}
		return len(pi) < len(pj)
	})
}
