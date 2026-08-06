package rules

type AtomPattern struct {
	From      string
	Predicate string
	To        string
}

type Rule struct {
	Name string
	If   []AtomPattern
	Then AtomPattern
}

func (p AtomPattern) vars() []string {
	out := make([]string, 0, 3)
	if isVar(p.From) {
		out = append(out, p.From)
	}
	if isVar(p.Predicate) {
		out = append(out, p.Predicate)
	}
	if isVar(p.To) {
		out = append(out, p.To)
	}
	return out
}

func isVar(s string) bool {
	return len(s) >= 2 && s[0] == '?' && s[1] != 0
}
