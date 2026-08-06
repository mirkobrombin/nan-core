package rules

type Bindings map[string]string

func (b Bindings) clone() Bindings {
	out := make(Bindings, len(b))
	for k, v := range b {
		out[k] = v
	}
	return out
}

func unifyField(pattern string, value string, binds Bindings) (Bindings, bool) {
	if isVar(pattern) {
		if existing, ok := binds[pattern]; ok {
			return binds, existing == value
		}
		out := binds.clone()
		out[pattern] = value
		return out, true
	}
	return binds, pattern == value
}

func (b Bindings) Apply(p AtomPattern) (AtomPattern, bool) {
	out := AtomPattern{From: p.From, Predicate: p.Predicate, To: p.To}
	fields := []*string{&out.From, &out.Predicate, &out.To}
	for _, f := range fields {
		if isVar(*f) {
			v, ok := b[*f]
			if !ok {
				return AtomPattern{}, false
			}
			*f = v
		}
	}
	return out, true
}
