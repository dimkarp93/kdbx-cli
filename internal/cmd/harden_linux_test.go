package cmd

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestHardenProcessClearsDumpable(t *testing.T) {
	hardenProcess()
	got, err := unix.PrctlRetInt(unix.PR_GET_DUMPABLE, 0, 0, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got != 0 {
		t.Errorf("dumpable: got %d, want 0", got)
	}
}
