package model

type Set map[string]struct{}

func NewSet(values ...string) Set {
	s := make(Set, len(values))
	s.Add(values...)
	return s
}

func (s Set) Add(values ...string) {
	for _, v := range values {
		s[v] = struct{}{}
	}
}

func (s Set) Contains(value string) bool {
	_, ok := s[value]
	return ok
}

func (s Set) Values() []string {
	values := make([]string, 0, len(s))
	for k := range s {
		values = append(values, k)
	}
	return values
}
