package belief

import (
	"errors"
	"sort"
	"sync"
)

type Store struct {
	mu sync.RWMutex

	beliefsByAtom    map[Atom]map[Polarity][]Belief
	resolutionByAtom map[Atom]Resolution
	count            int
}

func NewStore() *Store {
	return &Store{
		beliefsByAtom:    make(map[Atom]map[Polarity][]Belief),
		resolutionByAtom: make(map[Atom]Resolution),
	}
}

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.count
}

// Add inserts a belief. If the opposite polarity exists for the same atom, it returns a Contradiction.
func (s *Store) Add(b Belief) (*Contradiction, error) {
	if b.Atom.From == "" || b.Atom.To == "" {
		return nil, errors.New("from/to must be non-empty")
	}
	if b.Atom.Predicate == "" {
		return nil, errors.New("predicate must be non-empty")
	}
	if b.Source == "" {
		return nil, errors.New("source must be non-empty")
	}
	if b.Polarity != PolarityNegative && b.Polarity != PolarityPositive {
		return nil, errors.New("invalid polarity")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	byPol, ok := s.beliefsByAtom[b.Atom]
	if !ok {
		byPol = make(map[Polarity][]Belief)
		s.beliefsByAtom[b.Atom] = byPol
	}

	// de-dup by (polarity, source, timestamp)
	for _, existing := range byPol[b.Polarity] {
		if existing.Source == b.Source && existing.UnixNano == b.UnixNano {
			return nil, nil
		}
	}

	// Resolution is authoritative for runtime truth evaluation.
	// We still append beliefs to keep an audit trail, but contradictions are evaluated against the resolution if present.
	if res, ok := s.resolutionByAtom[b.Atom]; ok {
		byPol[b.Polarity] = append(byPol[b.Polarity], b)
		s.count++
		s.sortLocked(b.Atom)
		if b.Polarity != res.Polarity {
			c := &Contradiction{
				Atom: b.Atom,
				Existing: Belief{
					Atom:     res.Atom,
					Polarity: res.Polarity,
					Source:   res.Source,
					UnixNano: res.UnixNano,
				},
				Incoming: b,
				Reason:   "conflicts with resolution",
				UnixNano: b.UnixNano,
			}
			return c, nil
		}
		return nil, nil
	}

	opp := PolarityPositive
	if b.Polarity == PolarityPositive {
		opp = PolarityNegative
	}
	if len(byPol[opp]) > 0 {
		existing := byPol[opp][0]
		c := &Contradiction{
			Atom:     b.Atom,
			Existing: existing,
			Incoming: b,
			Reason:   "opposite polarity for same atom",
			UnixNano: b.UnixNano,
		}
		byPol[b.Polarity] = append(byPol[b.Polarity], b)
		s.count++
		s.sortLocked(b.Atom)
		return c, nil
	}

	byPol[b.Polarity] = append(byPol[b.Polarity], b)
	s.count++
	s.sortLocked(b.Atom)
	return nil, nil
}

func (s *Store) sortLocked(atom Atom) {
	byPol := s.beliefsByAtom[atom]
	for pol := range byPol {
		sort.Slice(byPol[pol], func(i, j int) bool {
			if byPol[pol][i].UnixNano != byPol[pol][j].UnixNano {
				return byPol[pol][i].UnixNano < byPol[pol][j].UnixNano
			}
			return byPol[pol][i].Source < byPol[pol][j].Source
		})
	}
}

func (s *Store) SetResolution(r Resolution) error {
	if r.Atom.From == "" || r.Atom.To == "" {
		return errors.New("from/to must be non-empty")
	}
	if r.Atom.Predicate == "" {
		return errors.New("predicate must be non-empty")
	}
	if r.Source == "" {
		return errors.New("source must be non-empty")
	}
	if r.Polarity != PolarityNegative && r.Polarity != PolarityPositive {
		return errors.New("invalid polarity")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.resolutionByAtom[r.Atom] = r
	return nil
}

func (s *Store) Resolution(atom Atom) (Resolution, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.resolutionByAtom[atom]
	return r, ok
}

// Snapshot returns beliefs for an atom in deterministic order.
func (s *Store) Snapshot(atom Atom) []Belief {
	s.mu.RLock()
	defer s.mu.RUnlock()

	byPol, ok := s.beliefsByAtom[atom]
	if !ok {
		return nil
	}

	out := make([]Belief, 0, len(byPol[PolarityPositive])+len(byPol[PolarityNegative]))
	out = append(out, byPol[PolarityNegative]...)
	out = append(out, byPol[PolarityPositive]...)
	return out
}

func (s *Store) Atoms() []Atom {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Atom, 0, len(s.beliefsByAtom))
	for a := range s.beliefsByAtom {
		out = append(out, a)
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

func (s *Store) Resolutions() []Resolution {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Resolution, 0, len(s.resolutionByAtom))
	for _, r := range s.resolutionByAtom {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		ai := out[i].Atom
		aj := out[j].Atom
		if ai.From != aj.From {
			return ai.From < aj.From
		}
		if ai.Predicate != aj.Predicate {
			return ai.Predicate < aj.Predicate
		}
		return ai.To < aj.To
	})
	return out
}
