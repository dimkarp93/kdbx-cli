package cmd

import (
	"cmp"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/config"
	"github.com/dimkarp93/kdbx-cli/internal/deliver"
	"github.com/dimkarp93/kdbx-cli/internal/resolve"
	"github.com/dimkarp93/kdbx-cli/internal/secretpipe"
	"github.com/dimkarp93/kdbx-cli/internal/tmpl"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

func sortedMappings(secrets map[string]string) []resolve.Mapping {
	pairs := resolve.MappingsFromMap(secrets)
	slices.SortFunc(pairs, func(a, b resolve.Mapping) int {
		return cmp.Or(cmp.Compare(a.Env, b.Env), cmp.Compare(a.Name, b.Name))
	})
	return pairs
}

func describeSection(cfg config.Config, tool string) string {
	_, hasTool := cfg.Sections[tool]
	_, hasDefault := cfg.Sections["default"]
	switch {
	case tool == "default" && hasDefault:
		return `"default"`
	case hasTool && hasDefault:
		return fmt.Sprintf("%q (merged over \"default\")", tool)
	case hasTool:
		return fmt.Sprintf("%q", tool)
	case hasDefault:
		return fmt.Sprintf("\"default\" (no section for %q)", tool)
	default:
		return fmt.Sprintf("(no section for %q and no \"default\")", tool)
	}
}

func isShellSafe(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_./-:@%+=,", r)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.IndexFunc(s, func(r rune) bool { return !isShellSafe(r) }) < 0 {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func renderCommand(mappings []resolve.Mapping, files []string, child []string) string {
	envParts := utils.Map(mappings, func(m resolve.Mapping) string {
		return fmt.Sprintf("%s=<secret from %s>", m.Env, m.Name)
	})
	argParts := utils.Map(child, func(arg string) string {
		return shellQuote(maskFilePlaceholders(arg, files))
	})
	return strings.Join(slices.Concat(envParts, argParts), " ")
}

func maskFilePlaceholders(arg string, files []string) string {
	for _, name := range files {
		arg = strings.ReplaceAll(arg, deliver.Placeholder(name), fmt.Sprintf("<file with %s>", name))
	}
	return arg
}

func templateName(t config.Template) string {
	return t.Name
}

func printPlan(plan runPlan, flags runFlags, child []string) {
	out := os.Stdout
	res := plan.res
	fmt.Fprintln(out, "Dry run — the command will NOT be executed.")
	fmt.Fprintln(out)

	printPlanHeader(out, plan, flags)
	printPlanMappings(out, sortedMappings(res.Secrets))
	printPlanStdin(out, res)
	printPlanFiles(out, res.Files)
	printPlanTemplates(out, res.Templates)
	printPlanAskpass(out, res)

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Command:")
	fileLikeNames := slices.Concat(res.Files, utils.Map(res.Templates, templateName))
	fmt.Fprintf(out, "  %s\n", renderCommand(sortedMappings(res.Secrets), fileLikeNames, child))
}

func printPlanHeader(out io.Writer, plan runPlan, flags runFlags) {
	fmt.Fprintf(out, "Config file:  %s\n", plan.cfgPath)
	if _, err := os.Stat(plan.cfgPath); err != nil {
		fmt.Fprintln(out, "              (not found — using flags only)")
	}

	fmt.Fprintf(out, "Tool:         %s\n", plan.tool)
	fmt.Fprintf(out, "Section:      %s\n", describeSection(plan.cfg, plan.tool))

	fmt.Fprintf(out, "Key-store:    %s\n", cmp.Or(plan.res.KeyStore, "(not configured)"))
	if flags.keyStore != "" {
		fmt.Fprintln(out, "              (from --key-store)")
	}
}

func printPlanMappings(out io.Writer, mappings []resolve.Mapping) {
	fmt.Fprintln(out)
	if len(mappings) == 0 {
		fmt.Fprintln(out, "Mappings:     (none configured)")
		return
	}

	fmt.Fprintln(out, "Mappings (env ← secret):")
	width := slices.Max(utils.Map(mappings, func(m resolve.Mapping) int { return len(m.Env) }))
	for _, m := range mappings {
		fmt.Fprintf(out, "  %-*s ← %s\n", width, m.Env, m.Name)
	}
}

func printPlanStdin(out io.Writer, res resolve.Resolved) {
	if len(res.Stdin) == 0 {
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Stdin (one line per secret, in order):")
	for _, name := range res.Stdin {
		fmt.Fprintf(out, "  <secret from %s>\n", name)
	}
	fmt.Fprintln(out, stdinAfterwards(res.StdinKeepOpen))
}

func stdinAfterwards(keepOpen bool) string {
	if keepOpen {
		return "  (stdin stays open afterwards)"
	}
	return "  (stdin is closed afterwards)"
}

func printPlanFiles(out io.Writer, files []string) {
	if len(files) == 0 {
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Files (placeholder ← secret):")
	for _, name := range files {
		fmt.Fprintf(out, "  %s ← %s\n", deliver.Placeholder(name), name)
	}
}

func printPlanTemplates(out io.Writer, templates []config.Template) {
	if len(templates) == 0 {
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Templates (placeholder ← rendered file):")
	for _, t := range templates {
		fmt.Fprintf(out, "  %s ← %s  %s\n", deliver.Placeholder(t.Name), t.Path, templateUsage(t.Path))
	}
}

func templateUsage(path string) string {
	titles, err := tmpl.Titles(path)
	if err != nil {
		return "(template file not found)"
	}
	return "(uses: " + strings.Join(titles, ", ") + ")"
}

func printPlanAskpass(out io.Writer, res resolve.Resolved) {
	if res.Askpass == "" {
		return
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Askpass:      %s\n", res.Askpass)
	fmt.Fprintf(out, "              handed out at most %d time(s)\n", secretpipe.AskpassReads(res.AskpassReads))
	fmt.Fprintf(out, "              %s\n", strings.Join(deliver.AskpassEnv, ", "))
}
