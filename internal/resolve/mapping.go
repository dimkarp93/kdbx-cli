package resolve

import (
	"cmp"
	"regexp"
	"slices"
)

type Mapping struct {
	Name string
	Env  string
}

func compareMappings(a, b Mapping) int {
	return cmp.Or(cmp.Compare(a.Name, b.Name), cmp.Compare(a.Env, b.Env))
}
func SortMappings(pairs []Mapping) {
	slices.SortFunc(pairs, compareMappings)
}
func MappingsFromMap(envToName map[string]string) []Mapping {
	out := make([]Mapping, 0, len(envToName))
	for env, name := range envToName {
		out = append(out, Mapping{Name: name, Env: env})
	}
	SortMappings(out)
	return out
}
func MappingsToMap(pairs []Mapping) map[string]string {
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		m[p.Env] = p.Name
	}
	return m
}
func MappingName(m Mapping) string {
	return m.Name
}

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isInvalidEnvName(env string) bool {
	panic("not implemented")
}
func ValidateEnvNames(envToName map[string]string) error {
	panic("not implemented")
}
