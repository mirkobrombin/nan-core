package kg

type NodeID string

type Edge struct {
	From      NodeID
	Predicate string
	To        NodeID
}

type Graph interface {
	AddFact(from NodeID, predicate string, to NodeID) error
	EdgesFrom(from NodeID) []Edge
	EdgesTo(to NodeID) []Edge
	EdgesAll() []Edge
	FactsCount() int
}
