//go:build !linux

package secretpipe

import (
	"os"
	"path/filepath"
)

func createAnonFile(name string) (*os.File, error) {
	dir, err := os.MkdirTemp("", "kdbx-cli-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, err
	}
	if err := os.Remove(path); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}
