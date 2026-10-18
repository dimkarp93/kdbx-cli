package store

import (
	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
)

type storeState struct {
	path     string
	exists   bool
	password string
	missing  []string
}

func ReconcileStores(byStore map[string][]string, assumeYes bool, cache keyring.Cache) {
	panic("not implemented")
}
func gatherMissing(byStore map[string][]string, cache keyring.Cache) []storeState {
	panic("not implemented")
}
func inspectStore(path string, titles []string, cache keyring.Cache) (storeState, bool) {
	panic("not implemented")
}
func missingTitles(entries []keepass.Entry, titles []string) []string {
	panic("not implemented")
}
func totalMissing(states []storeState) int {
	panic("not implemented")
}
func printMissing(states []storeState) {
	panic("not implemented")
}
func missingStoreSuffix(exists bool) string {
	panic("not implemented")
}
func fillStore(s storeState, assumeYes bool, cache keyring.Cache) {
	panic("not implemented")
}
func openForFill(s storeState, assumeYes bool, cache keyring.Cache) (string, bool) {
	panic("not implemented")
}
func createStore(s storeState, assumeYes bool, cache keyring.Cache) (string, bool) {
	panic("not implemented")
}
func readNewPassword(path string, read func(prompt string) (string, error)) (string, error) {
	panic("not implemented")
}
