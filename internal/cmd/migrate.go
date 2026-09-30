package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dimkarp93/kdbx-cli/internal/config"
)

func parseMigrateArgs(args []string) (path string, from, to int) {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&path, "config", "", "")
	fs.IntVar(&from, "from", 0, "")
	fs.IntVar(&to, "to", config.CurrentVersion, "")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "invalid arguments for migrate: %v\n", args)
		os.Exit(2)
	}
	return path, from, to
}

func cmdMigrate(path string, from, to int) {
	if path == "" {
		path = config.DefaultPath()
	}
	err := config.Migrate(path, from, to)
	if errors.Is(err, config.ErrAlreadyMigrated) {
		fmt.Printf("Config %s is already at version %d\n", path, to)
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot migrate config:", err)
		os.Exit(1)
	}
	fmt.Printf("Config %s migrated from version %d to %d\n", path, from, to)
}
