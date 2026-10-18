package resolve

import (
	"regexp"
)

type Mapping struct {
	Name string
	Env  string
}

func compareMappings(a, b Mapping) int {
	panic("not implemented")
}
func SortMappings(pairs []Mapping) {
	panic("not implemented")
}
func MappingsFromMap(envToName map[string]string) []Mapping {
	panic("not implemented")
}
func MappingsToMap(pairs []Mapping) map[string]string {
	panic("not implemented")
}
func MappingName(m Mapping) string {
	panic("not implemented")
}

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isInvalidEnvName(env string) bool {
	panic("not implemented")
}
func ValidateEnvNames(envToName map[string]string) error {
	panic("not implemented")
}
