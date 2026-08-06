package engine

import "github.com/mirkobrombin/nan-core/pkg/ilp"

func (e *Engine) ILPSuggest(minChains int) []ilp.Suggestion {
	return ilp.SuggestTransitive(e.graph.EdgesAll(), minChains)
}
