package tui

import (
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

type pairList struct {
	pairs  []resolve.Mapping
	cursor int
}

func (l pairList) moveCursor(delta int) pairList {
	panic("not implemented")
}
func (l pairList) deleteSelected() pairList {
	panic("not implemented")
}
func (l pairList) at(index int) resolve.Mapping {
	panic("not implemented")
}
func (l pairList) envTaken(env string, except int) bool {
	panic("not implemented")
}
func (l pairList) upsert(index int, pair resolve.Mapping) pairList {
	panic("not implemented")
}
func (l pairList) view(focused bool) string {
	panic("not implemented")
}
