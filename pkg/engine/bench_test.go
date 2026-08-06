package engine

import (
	"fmt"
	"testing"

	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/kg"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

func BenchmarkIngest(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("n=%d", n), func(b *testing.B) {
			for iter := 0; iter < b.N; iter++ {
				e, _ := New()
				for i := 0; i < n; i++ {
					from := fmt.Sprintf("node%d", i)
					to := fmt.Sprintf("node%d", i+1)
					_, _ = e.Ingest(belief.Belief{
						Atom:     belief.Atom{From: kg.NodeID(from), Predicate: "knows", To: kg.NodeID(to)},
						Polarity: belief.PolarityPositive,
						Source:   "bench",
					})
				}
			}
		})
	}
}

func BenchmarkEvaluate(b *testing.B) {
	e, _ := New()
	for i := 0; i < 1000; i++ {
		from := fmt.Sprintf("node%d", i)
		to := fmt.Sprintf("node%d", i+1)
		_, _ = e.Ingest(belief.Belief{
			Atom:     belief.Atom{From: kg.NodeID(from), Predicate: "knows", To: kg.NodeID(to)},
			Polarity: belief.PolarityPositive,
			Source:   "bench",
		})
	}

	target := belief.Atom{From: "node500", Predicate: "knows", To: "node501"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Evaluate(target)
	}
}

func BenchmarkEvaluateWithRules(b *testing.B) {
	e, _ := New()
	_ = e.AddRule(rules.Rule{
		Name: "trans",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "knows", To: "?y"},
			{From: "?y", Predicate: "knows", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "knows", To: "?z"},
	})
	for i := 0; i < 50; i++ {
		from := fmt.Sprintf("n%d", i)
		to := fmt.Sprintf("n%d", i+1)
		_, _ = e.Ingest(belief.Belief{
			Atom:     belief.Atom{From: kg.NodeID(from), Predicate: "knows", To: kg.NodeID(to)},
			Polarity: belief.PolarityPositive,
			Source:   "bench",
		})
	}

	target := belief.Atom{From: "n0", Predicate: "knows", To: "n10"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Evaluate(target)
	}
}

func BenchmarkExplainGraph(b *testing.B) {
	e, _ := New()
	_ = e.AddRule(rules.Rule{
		Name: "trans",
		If: []rules.AtomPattern{
			{From: "?x", Predicate: "trusts", To: "?y"},
			{From: "?y", Predicate: "trusts", To: "?z"},
		},
		Then: rules.AtomPattern{From: "?x", Predicate: "trusts", To: "?z"},
	})
	for i := 0; i < 20; i++ {
		from := fmt.Sprintf("p%d", i)
		to := fmt.Sprintf("p%d", i+1)
		_, _ = e.Ingest(belief.Belief{
			Atom:     belief.Atom{From: kg.NodeID(from), Predicate: "trusts", To: kg.NodeID(to)},
			Polarity: belief.PolarityPositive,
			Source:   "bench",
		})
	}

	target := belief.Atom{From: "p0", Predicate: "trusts", To: "p5"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.ExplainGraph(target, 3)
	}
}
