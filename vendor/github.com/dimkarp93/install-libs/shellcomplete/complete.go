package shellcomplete

import (
	"slices"
	"strconv"
	"strings"
)

const CompleteCommand = "__complete"

const (
	DirNone    = ":none"
	DirDirs    = ":dirs"
	DirDefault = ":default"
	DirCommand = ":command"
)

const (
	CompletionCommand = "completion"
	InstallCommand    = "install-completions"
	UninstallCommand  = "uninstall-completions"
)

var (
	Shells  = []string{"bash", "zsh"}
	Targets = []string{"bash", "zsh", "all"}
)

type Flag struct {
	Name   string
	Bool   bool
	Values func() []string
	Dirs   bool
	Files  bool
}

type Command struct {
	Name       string
	Flags      []Flag
	Args       func(pos int) []string
	Intermixed bool
	Hidden     bool
}

type Spec struct {
	Bin              string
	Flags            []Flag
	Args             func(pos int) []string
	Intermixed       bool
	Commands         []Command
	CommandAfterDash bool
}

func Static(items ...string) func() []string {
	return func() []string { return items }
}

func Positional(items ...string) func(pos int) []string {
	return func(pos int) []string {
		if pos == 0 {
			return items
		}
		return []string{DirNone}
	}
}

func Anything(int) []string {
	return []string{DirDefault}
}

func builtins() []Command {
	dry := []Flag{{Name: "--dry-run", Bool: true}}
	return []Command{
		{Name: CompletionCommand, Args: Positional(Shells...)},
		{Name: InstallCommand, Flags: dry, Args: Positional(Targets...), Intermixed: true},
		{Name: UninstallCommand, Flags: dry, Args: Positional(Targets...), Intermixed: true},
	}
}

func (s Spec) commands() []Command {
	return append(slices.Clone(s.Commands), builtins()...)
}

func (s Spec) lookup(name string) (Command, bool) {
	for _, c := range s.commands() {
		if c.Name == name {
			return c, true
		}
	}
	return Command{}, false
}

func (s Spec) Complete(words []string) []string {
	raw := words
	words = joinColons(joinEquals(words))
	if len(words) == 0 {
		words = []string{""}
	}
	cur := words[len(words)-1]
	done := words[:len(words)-1]

	root := Command{Flags: s.Flags, Args: s.Args, Intermixed: s.Intermixed}
	sc := scan(root.Flags, done, false)
	if sc.dashed {
		if s.CommandAfterDash {
			return commandOffset(raw, done, sc.dash)
		}
		return complete(root, done, cur)
	}
	if len(sc.free) > 0 {
		if c, ok := s.lookup(sc.free[0]); ok && (len(s.Commands) > 0 || sc.first == 0) {
			return complete(c, done[sc.first+1:], cur)
		}
		if len(s.Commands) > 0 {
			return []string{DirNone}
		}
	}
	if len(s.Commands) > 0 {
		if out, ok := flagCompletion(root.Flags, done, cur); ok {
			return out
		}
		var names []string
		for _, c := range s.commands() {
			if !c.Hidden {
				names = append(names, c.Name)
			}
		}
		return filter(names, cur)
	}
	return complete(root, done, cur)
}

func commandOffset(raw, done []string, dash int) []string {
	ordinal := countDashes(done[:dash])
	start := nthDash(raw, ordinal) + 1
	return []string{DirCommand, strconv.Itoa(start)}
}

func countDashes(words []string) int {
	n := 0
	for _, w := range words {
		if w == "--" {
			n++
		}
	}
	return n
}

func nthDash(words []string, n int) int {
	for i, w := range words {
		if w != "--" {
			continue
		}
		if n == 0 {
			return i
		}
		n--
	}
	return -1
}

func complete(c Command, rest []string, cur string) []string {
	sc := scan(c.Flags, rest, c.Intermixed)
	if !sc.stopped {
		if out, ok := flagCompletion(c.Flags, rest, cur); ok {
			return out
		}
	}
	if c.Args == nil {
		return []string{DirNone}
	}
	return filter(c.Args(len(sc.free)), cur)
}

func flagCompletion(flags []Flag, rest []string, cur string) ([]string, bool) {
	if len(rest) > 0 {
		prev := rest[len(rest)-1]
		if isFlagWord(prev) && !strings.Contains(prev, "=") {
			if f, ok := find(flags, prev); ok && !f.Bool {
				return values(f, cur), true
			}
		}
	}
	if !strings.HasPrefix(cur, "-") {
		return nil, false
	}
	if name, value, ok := strings.Cut(cur, "="); ok {
		f, found := find(flags, name)
		if !found || f.Bool {
			return []string{DirNone}, true
		}
		out := values(f, value)
		for i, v := range out {
			if !isDirective(v) {
				out[i] = name + "=" + v
			}
		}
		return out, true
	}
	return flagNames(flags, cur), true
}

func values(f Flag, prefix string) []string {
	switch {
	case f.Values != nil:
		return filter(f.Values(), prefix)
	case f.Dirs:
		return []string{DirDirs}
	case f.Files:
		return []string{DirDefault}
	}
	return []string{DirNone}
}

func flagNames(flags []Flag, cur string) []string {
	var names []string
	for _, f := range flags {
		name := f.Name
		if strings.HasPrefix(cur, "--") && !strings.HasPrefix(name, "--") && len(name) > 2 {
			name = "-" + name
		}
		if !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	out := filter(names, cur)
	if len(out) == 0 {
		return []string{DirNone}
	}
	return out
}

type scanned struct {
	free    []string
	stopped bool
	dashed  bool
	first   int
	dash    int
}

func scan(flags []Flag, args []string, intermixed bool) scanned {
	var free []string
	stopped, dashed := false, false
	first, dash := -1, -1
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if stopped {
			free = append(free, arg)
			continue
		}
		if arg == "--" {
			stopped, dashed = true, true
			dash = i
			continue
		}
		if isFlagWord(arg) {
			if !strings.Contains(arg, "=") {
				if f, ok := find(flags, arg); ok && !f.Bool {
					i++
				}
			}
			continue
		}
		if first < 0 {
			first = i
		}
		free = append(free, arg)
		if !intermixed {
			stopped = true
		}
	}
	return scanned{
		free:    free,
		stopped: stopped,
		dashed:  dashed,
		first:   first,
		dash:    dash,
	}
}

func find(flags []Flag, word string) (Flag, bool) {
	name, _, _ := strings.Cut(strings.TrimLeft(word, "-"), "=")
	for _, f := range flags {
		if strings.TrimLeft(f.Name, "-") == name {
			return f, true
		}
	}
	return Flag{}, false
}

func joinEquals(words []string) []string {
	out := make([]string, 0, len(words))
	for i, w := range words {
		if w == "=" && len(out) > 0 {
			prev := out[len(out)-1]
			if isFlagWord(prev) && !strings.Contains(prev, "=") {
				if i == len(words)-1 {
					out = append(out, "")
				}
				continue
			}
		}
		out = append(out, w)
	}
	return out
}

func joinColons(words []string) []string {
	out := make([]string, 0, len(words))
	glue := false
	for _, w := range words {
		switch {
		case isColons(w) && len(out) > 0 && isValueWord(out[len(out)-1]):
			out[len(out)-1] += w
			glue = true
		case glue && isValueWord(w):
			out[len(out)-1] += w
			glue = false
		default:
			out = append(out, w)
			glue = false
		}
	}
	return out
}

func isColons(word string) bool {
	return word != "" && strings.Trim(word, ":") == ""
}

func isValueWord(word string) bool {
	return word != "--" && !isFlagWord(word)
}

func isFlagWord(arg string) bool {
	return strings.HasPrefix(arg, "-") && arg != "-" && arg != "--"
}

func isDirective(v string) bool {
	return v == DirNone || v == DirDirs || v == DirDefault
}

func filter(items []string, prefix string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if isDirective(item) {
			return []string{item}
		}
		if strings.HasPrefix(item, prefix) {
			out = append(out, item)
		}
	}
	return out
}
