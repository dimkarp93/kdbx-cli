package config

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func PermissionWarnings(path string) []string {
	var out []string
	for _, p := range []string{path, filepath.Dir(path)} {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		if fi.Mode().Perm()&0o022 != 0 {
			out = append(out, fmt.Sprintf("%s is writable by group or others", p))
		}
		if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Uid != 0 && int(st.Uid) != os.Getuid() {
			out = append(out, fmt.Sprintf("%s is owned by uid %d", p, st.Uid))
		}
	}
	return out
}

func WarnUnsafePermissions(path string) {
	for _, w := range PermissionWarnings(path) {
		fmt.Fprintf(os.Stderr, "warning: %s; whoever can change it decides where your secrets go\n", w)
	}
}
