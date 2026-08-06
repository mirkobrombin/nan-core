package engine

import (
	"sort"

	"github.com/mirkobrombin/nan-core/pkg/belief"
)

// SemanticQuery returns atoms semantically similar to the query atom,
// evaluated against the engine's VSA space with proof for each.
type SemanticResult struct {
	Atom       belief.Atom
	Similarity float64
	Truth      TruthValue
	Proof      Proof
}

// SemanticQuery finds atoms in the knowledge base that are semantically
// similar to the query atom, using the VSA hyperdimensional space.
// Results are sorted by similarity descending, deterministically.
func (e *Engine) SemanticQuery(atom belief.Atom, threshold float64) []SemanticResult {
	matches := e.semantic.Similar(atom, threshold)

	results := make([]SemanticResult, 0, len(matches))
	for _, m := range matches {
		v, pf := e.Evaluate(m.Atom)
		results = append(results, SemanticResult{
			Atom:       m.Atom,
			Similarity: m.Similarity,
			Truth:      v,
			Proof:      pf,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Similarity != results[j].Similarity {
			return results[i].Similarity > results[j].Similarity
		}
		k1 := atomKey(results[i].Atom)
		k2 := atomKey(results[j].Atom)
		return k1 < k2
	})

	return results
}
