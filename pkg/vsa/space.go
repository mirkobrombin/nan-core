package vsa

import (
	"hash/fnv"
	"sort"
	"sync"
)

const DefaultDims = 10000

// Space is a deterministic hyperdimensional concept space.
// Each symbol (string) is assigned a unique random vector seeded from its hash.
// Composite concepts are built via Bind (XOR) and Bundle (majority).
type Space struct {
	dims int
	mu   sync.RWMutex
	vecs map[string]Vector
}

func NewSpace(dims int) *Space {
	if dims <= 0 {
		dims = DefaultDims
	}
	return &Space{dims: dims, vecs: make(map[string]Vector)}
}

// Encode returns the deterministic vector for a symbol.
// Same symbol always produces the same vector.
func (s *Space) Encode(symbol string) Vector {
	s.mu.RLock()
	if v, ok := s.vecs[symbol]; ok {
		s.mu.RUnlock()
		return v
	}
	s.mu.RUnlock()

	seed := symbolSeed(symbol)
	v, _ := NewRandom(s.dims, seed)

	s.mu.Lock()
	s.vecs[symbol] = v
	s.mu.Unlock()
	return v
}

// EncodeTuple encodes a (from, predicate, to) triple as a single vector.
// Uses role-filler binding: Bind(from_role, from) XOR Bind(pred_role, pred) XOR Bind(to_role, to).
func (s *Space) EncodeTuple(from, predicate, to string) Vector {
	roleFrom := s.Encode("__role_from")
	rolePred := s.Encode("__role_pred")
	roleTo := s.Encode("__role_to")

	vFrom := s.Encode(from)
	vPred := s.Encode(predicate)
	vTo := s.Encode(to)

	bFrom, _ := BindXOR(roleFrom, vFrom)
	bPred, _ := BindXOR(rolePred, vPred)
	bTo, _ := BindXOR(roleTo, vTo)

	// XOR all three role-filler bindings
	r1, _ := BindXOR(bFrom, bPred)
	r2, _ := BindXOR(r1, bTo)
	return r2
}

// Nearest finds the top-k most similar symbols to a query vector.
func (s *Space) Nearest(query Vector, k int) []Match {
	s.mu.RLock()
	defer s.mu.RUnlock()

	matches := make([]Match, 0, len(s.vecs))
	for sym, v := range s.vecs {
		sim, err := Similarity(query, v)
		if err != nil {
			continue
		}
		matches = append(matches, Match{Symbol: sym, Similarity: sim})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Similarity != matches[j].Similarity {
			return matches[i].Similarity > matches[j].Similarity
		}
		return matches[i].Symbol < matches[j].Symbol
	})

	if k > len(matches) {
		k = len(matches)
	}
	return matches[:k]
}

// Match represents a similarity search result.
type Match struct {
	Symbol     string
	Similarity float64
}

func symbolSeed(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	v := h.Sum64()
	if v == 0 {
		v = 1
	}
	return v
}
