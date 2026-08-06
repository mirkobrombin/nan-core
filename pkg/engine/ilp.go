package engine

import "github.com/mirkobrombin/nan-core/pkg/ilp"

func (e *Engine) ILPAudit() []ilp.Example {
	out := make([]ilp.Example, len(e.ilpAudit))
	copy(out, e.ilpAudit)
	return out
}
