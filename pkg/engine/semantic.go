package engine

import (
	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/vsa"
)

// SemanticIndex maps atoms into a hyperdimensional VSA space,
// enabling approximate/fuzzy similarity queries beyond exact string matching.
type SemanticIndex struct {
	space  *vsa.Space
	tuples map[belief.Atom]vsa.Vector
}

func newSemanticIndex(dims int) *SemanticIndex {
	return &SemanticIndex{
		space:  vsa.NewSpace(dims),
		tuples: make(map[belief.Atom]vsa.Vector),
	}
}

// Index adds an atom to the semantic space.
func (si *SemanticIndex) Index(atom belief.Atom) {
	if _, ok := si.tuples[atom]; ok {
		return
	}
	v := si.space.EncodeTuple(string(atom.From), atom.Predicate, string(atom.To))
	si.tuples[atom] = v
}

// Similar returns atoms semantically similar to the query atom, with similarity scores.
func (si *SemanticIndex) Similar(atom belief.Atom, threshold float64) []SemanticMatch {
	qv := si.space.EncodeTuple(string(atom.From), atom.Predicate, string(atom.To))
	out := make([]SemanticMatch, 0, 16)
	for a, v := range si.tuples {
		sim, err := vsa.Similarity(qv, v)
		if err != nil {
			continue
		}
		if sim >= threshold {
			out = append(out, SemanticMatch{Atom: a, Similarity: sim})
		}
	}
	return out
}

// SemanticMatch represents an atom with its similarity score.
type SemanticMatch struct {
	Atom       belief.Atom
	Similarity float64
}
