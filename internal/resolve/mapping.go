package resolve

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/utils"
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
	return !envNamePattern.MatchString(env)
}

func ValidateEnvNames(envToName map[string]string) error {
	invalid := utils.Map(utils.Filter(utils.SortedKeys(envToName), isInvalidEnvName), strconv.Quote)
	if len(invalid) == 0 {
		return nil
	}

	slices.Sort(invalid)
	return fmt.Errorf("invalid env variable name(s): %s", strings.Join(invalid, ", "))
}
