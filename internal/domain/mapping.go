package domain

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Mapping struct {
	Name string
	Env  string
}

func SortMappings(pairs []Mapping) {
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Name != pairs[j].Name {
			return pairs[i].Name < pairs[j].Name
		}
		return pairs[i].Env < pairs[j].Env
	})
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

var envNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func ValidateEnvNames(envToName map[string]string) error {
	var invalid []string
	for env := range envToName {
		if !envNamePattern.MatchString(env) {
			invalid = append(invalid, fmt.Sprintf("%q", env))
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	sort.Strings(invalid)
	return fmt.Errorf("invalid env variable name(s): %s", strings.Join(invalid, ", "))
}

const (
	ChannelStdin    = "(stdin)"
	ChannelFile     = "(file)"
	ChannelTemplate = "(template)"
	ChannelAskpass  = "(askpass)"
)
