package config

import (
	"os"
	"path/filepath"

	"github.com/dimkarp93/install-libs/xdgpath"
)

const dirName = "kdbx-cli"

func ExpandHome(p string) string {
	return xdgpath.ExpandHome(p)
}

func Dir() string {
	return xdgpath.ConfigDir(dirName)
}

func DefaultPath() string {
	primary := filepath.Join(Dir(), "default")
	home, err := os.UserHomeDir()
	if err != nil {
		return primary
	}
	legacy := filepath.Join(home, ".config", dirName, "default")
	return xdgpath.WithLegacy(primary, legacy)
}
