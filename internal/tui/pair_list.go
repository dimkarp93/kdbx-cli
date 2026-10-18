package tui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/resolve"
)

type pairList struct {
	pairs  []resolve.Mapping
	cursor int
}

func (l pairList) moveCursor(delta int) pairList {
	next := l.cursor + delta
	if next >= 0 && next < len(l.pairs) {
		l.cursor = next
	}
	return l
}

func (l pairList) deleteSelected() pairList {
	if len(l.pairs) == 0 {
		return l
	}

	l.pairs = slices.Delete(l.pairs, l.cursor, l.cursor+1)
	l.cursor = min(l.cursor, max(len(l.pairs)-1, 0))
	return l
}

func (l pairList) at(index int) resolve.Mapping {
	if index < 0 {
		return resolve.Mapping{}
	}
	return l.pairs[index]
}

func (l pairList) envTaken(env string, except int) bool {
	for i, p := range l.pairs {
		if p.Env == env && i != except {
			return true
		}
	}
	return false
}

func (l pairList) upsert(index int, pair resolve.Mapping) pairList {
	var pairs []resolve.Mapping
	if index >= 0 {
		pairs = slices.Clone(l.pairs)
		pairs[index] = pair
	} else {
		pairs = slices.Concat(l.pairs, []resolve.Mapping{pair})
	}
	resolve.SortMappings(pairs)

	return pairList{pairs: pairs, cursor: slices.Index(pairs, pair)}
}

func (l pairList) view(focused bool) string {
	var b strings.Builder
	b.WriteString(labelStyle.Render(sectionLabel("secrets (name → env)", focused)) + "\n")
	if len(l.pairs) == 0 {
		b.WriteString(hintStyle.Render("    (none — press 'a' to add)") + "\n")
		return b.String()
	}

	for i, p := range l.pairs {
		b.WriteString(listLine(fmt.Sprintf("%s → %s", p.Name, p.Env), focused && i == l.cursor))
	}
	return b.String()
}
