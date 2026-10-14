package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/keyring"
)

func cmdLoad(flags runFlags) int {
	panic("not implemented")
}

func printLoaded(keyStore string, cache keyring.Cache) {
	granted := cache.Granted(keyStore)
	sort.Strings(granted)
	fmt.Printf("Cache enabled for %s\n", keyStore)
	fmt.Printf("  titles: %s\n", strings.Join(granted, ", "))
	if st := cache.Status(keyStore); st.Cached {
		fmt.Printf("  expires in %s\n", formatDuration(st.ExpiresIn))
	}
}
