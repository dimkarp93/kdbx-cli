package store

import (
	"fmt"
	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
	"os"
	"slices"
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
	var states []storeState
	for _, ks := range utils.SortedKeys(byStore) {
		titles := byStore[ks]
		if len(titles) == 0 {
			continue
		}
		if s, ok := inspectStore(ks, titles, cache); ok {
			states = append(states, s)
		}
	}
	return states
}
func inspectStore(path string, titles []string, cache keyring.Cache) (storeState, bool) {
	if !fileExists(path) {
		return storeState{path: path, missing: slices.Clone(titles)}, true
	}
	out, pw, err := UnlockExport(path, cache, fmt.Sprintf("Enter password for %s: ", path), true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", path, err)
		return storeState{}, false
	}
	entries, _ := keepass.ParseSecrets(out)
	return storeState{path: path, exists: true, password: pw, missing: missingTitles(entries, titles)}, true
}
func missingTitles(entries []keepass.Entry, titles []string) []string {
	return utils.Filter(titles, func(title string) bool {
		_, err := keepass.LookupSecret(entries, title)
		return err != nil
	})
}
func totalMissing(states []storeState) int {
	total := 0
	for _, s := range states {
		total += len(s.missing)
	}
	return total
}
func printMissing(states []storeState) {
	fmt.Println("Missing secrets:")
	for _, s := range states {
		if len(s.missing) == 0 {
			continue
		}
		fmt.Printf("  %s%s\n", s.path, missingStoreSuffix(s.exists))
		for _, t := range s.missing {
			fmt.Printf("    - %s\n", t)
		}
	}
}
func missingStoreSuffix(exists bool) string {
	if exists {
		return ""
	}
	return "  (key-store does not exist)"
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
