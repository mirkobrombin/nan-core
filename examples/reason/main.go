// Command reason shows the core in isolation: record two facts and a rule,
// then ask a question the rule has to derive, with its proof.
//
//	go run ./examples/reason
package main

import (
	"fmt"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/engine"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

func main() {
	e, err := engine.New()
	if err != nil {
		panic(err)
	}

	// Two facts.
	fact := func(from, pred, to string) {
		_, err := e.Ingest(belief.Belief{
			Atom:     belief.Atom{From: kg.NodeID(from), Predicate: pred, To: kg.NodeID(to)},
			Polarity: belief.PolarityPositive,
			Source:   "example",
		})
		if err != nil {
			panic(err)
		}
	}
	fact("socrate", "isa", "uomo")
	fact("uomo", "isa", "mortale")

	// A transitive rule: X isa Y and Y isa Z gives X isa Z.
	err = e.AddRule(rules.Rule{
		Name: "isa-transitive",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "isa", To: "?y"},
			{From: "?y", Predicate: "isa", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "isa", To: "?z"},
	})
	if err != nil {
		panic(err)
	}

	// A fact the engine was never told, only the rule can reach it.
	q := belief.Atom{From: "socrate", Predicate: "isa", To: "mortale"}
	v, proof := e.Evaluate(q)

	fmt.Printf("socrate isa mortale? truth=%d proof=%s\n", v, proof.Kind)
}
