package cmd

import "golang.org/x/sys/unix"

func hardenProcess() {
	_ = unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)
}
