package store

import (
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
)

func UnlockExport(keyStore string, cache keyring.Cache, prompt string, allowCache bool) (string, string, error) {
	panic("not implemented")
}
func exportCached(keyStore string, cache keyring.Cache) (string, string, bool) {
	panic("not implemented")
}
func exportPrompted(keyStore string, cache keyring.Cache, prompt string) (string, string, error) {
	panic("not implemented")
}
func export(keyStore, password string) (string, error) {
	panic("not implemented")
}
