package ilp

import (
	"github.com/mirkobrombin/nan-core/pkg/belief"
	"github.com/mirkobrombin/nan-core/pkg/rules"
)

type Example struct {
	Belief belief.Belief
}

type Suggestion struct {
	Rule   rules.Rule
	Reason string
}

type Inducer interface {
	Observe(ex Example) []Suggestion
}

type NoopInducer struct{}

func (NoopInducer) Observe(ex Example) []Suggestion { return nil }
