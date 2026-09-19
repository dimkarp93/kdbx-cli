package secretpipe

import (
	"os"

	"golang.org/x/sys/unix"
)

func createAnonFile(name string) (*os.File, error) {
	fd, err := unix.MemfdCreate("kdbx-cli-"+name, unix.MFD_CLOEXEC)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "/dev/fd/"+name), nil
}
