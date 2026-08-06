package belief

import "github.com/mirkobrombin/nan-core/pkg/kg"

type Polarity uint8

const (
	PolarityNegative Polarity = 0
	PolarityPositive Polarity = 1
)

type Atom struct {
	From      kg.NodeID
	Predicate string
	To        kg.NodeID
}

type Belief struct {
	Atom     Atom
	Polarity Polarity
	Source   string
	UnixNano int64
}

type Resolution struct {
	Atom     Atom
	Polarity Polarity
	Source   string
	UnixNano int64
}

type Contradiction struct {
	Atom     Atom
	Existing Belief
	Incoming Belief
	Reason   string
	UnixNano int64
}
