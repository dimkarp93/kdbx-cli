package secretpipe

import (
	"os"

	"golang.org/x/sys/unix"
)

func createAnonFile(name string) (*os.File, error) {
	fd, err := unix.MemfdCreate("kdbx-cli-"+name, unix.MFD_CLOEXEC|unix.MFD_ALLOW_SEALING)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), "/dev/fd/"+name), nil
}

func sealAnonFile(f *os.File) error {
	_, err := unix.FcntlInt(f.Fd(), unix.F_ADD_SEALS, unix.F_SEAL_SEAL|unix.F_SEAL_SHRINK|unix.F_SEAL_GROW|unix.F_SEAL_WRITE)
	return err
}
