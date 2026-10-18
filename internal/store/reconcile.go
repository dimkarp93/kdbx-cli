package store

import (
	"fmt"
	"os"
	"slices"

	"github.com/dimkarp93/kdbx-cli/internal/keepass"
	"github.com/dimkarp93/kdbx-cli/internal/keyring"
	"github.com/dimkarp93/kdbx-cli/internal/term"
	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

type storeState struct {
	path     string
	exists   bool
	password string
	missing  []string
}

func ReconcileStores(byStore map[string][]string, assumeYes bool, cache keyring.Cache) {
	keepass.CheckEngine()
	states := gatherMissing(byStore, cache)
	if totalMissing(states) == 0 {
		fmt.Println("All secrets are present in the key-store(s).")
		return
	}

	printMissing(states)

	for _, s := range states {
		if len(s.missing) > 0 {
			fillStore(s, assumeYes, cache)
		}
	}
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
	password, ok := openForFill(s, assumeYes, cache)
	if !ok {
		return
	}

	for _, t := range s.missing {
		if err := keepass.AddEmptySecret(s.path, password, t); err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Printf("  + %s → %s\n", t, s.path)
	}
}

func openForFill(s storeState, assumeYes bool, cache keyring.Cache) (string, bool) {
	if !s.exists {
		return createStore(s, assumeYes, cache)
	}
	if !term.Confirm(fmt.Sprintf("Add %d empty secret(s) to %s?", len(s.missing), s.path), assumeYes) {
		fmt.Println("Skipped", s.path)
		return "", false
	}
	return s.password, true
}

func createStore(s storeState, assumeYes bool, cache keyring.Cache) (string, bool) {
	if !term.Confirm(fmt.Sprintf("Create key-store %s and add %d empty secret(s)?", s.path, len(s.missing)), assumeYes) {
		fmt.Println("Skipped", s.path)
		return "", false
	}

	pw, err := readNewPassword(s.path, term.ReadPassword)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Println("Skipped", s.path)
		return "", false
	}

	if err := keepass.CreateStore(s.path, pw); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return "", false
	}

	cache.Remember(s.path, pw)
	fmt.Println("Created key-store", s.path)
	return pw, true
}

func readNewPassword(path string, read func(prompt string) (string, error)) (string, error) {
	pw, err := read("Set a password for the new key-store " + path + ": ")
	if err != nil {
		return "", err
	}
	if pw == "" {
		return "", fmt.Errorf("password for %s cannot be empty", path)
	}

	again, err := read("Repeat the password for " + path + ": ")
	if err != nil {
		return "", err
	}
	if again != pw {
		return "", fmt.Errorf("passwords for %s do not match", path)
	}
	return pw, nil
}
