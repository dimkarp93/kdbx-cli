package config

import (
	"os"
	"path/filepath"
	"strings"
)

const dirName = "kdbx-cli"

func ExpandHome(p string) string {
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", dirName)
}

func DefaultPath() string {
	return filepath.Join(Dir(), "default")
}
