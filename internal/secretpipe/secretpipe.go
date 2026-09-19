package secretpipe

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Set struct {
	dir  string
	wg   sync.WaitGroup
	done chan struct{}
	once sync.Once
}

func NewSet() *Set {
	return &Set{done: make(chan struct{})}
}

func (s *Set) Dir() string { return s.dir }

// The secret lives in an anonymous in-memory file; the child reads it back through
// /dev/fd/N, so the value never gets a name on any filesystem.
func (s *Set) File(name, value string) (*os.File, error) {
	f, err := createAnonFile(sanitize(name))
	if err != nil {
		return nil, fmt.Errorf("cannot create in-memory file for %q: %w", name, err)
	}
	if _, err := f.WriteString(value); err != nil {
		f.Close()
		return nil, err
	}
	if _, err := f.Seek(0, 0); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func (s *Set) Askpass(name, value string) (string, error) {
	if err := s.makeDir(); err != nil {
		return "", err
	}
	fifo := filepath.Join(s.dir, "secret")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		return "", fmt.Errorf("cannot create fifo for %q: %w", name, err)
	}
	script := filepath.Join(s.dir, "askpass.sh")
	body := "#!/bin/sh\nexec head -n 1 " + shellQuote(fifo) + "\n"
	if err := os.WriteFile(script, []byte(body), 0700); err != nil {
		return "", err
	}
	s.wg.Add(1)
	go s.feed(fifo, value+"\n")
	return script, nil
}

func (s *Set) Close() {
	s.once.Do(func() { close(s.done) })
	s.wg.Wait()
	if s.dir != "" {
		os.RemoveAll(s.dir)
	}
}

func (s *Set) makeDir() error {
	if s.dir != "" {
		return nil
	}
	base := os.Getenv("XDG_RUNTIME_DIR")
	if fi, err := os.Stat(base); base == "" || err != nil || !fi.IsDir() {
		base = ""
	}
	dir, err := os.MkdirTemp(base, "kdbx-cli-")
	if err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		os.RemoveAll(dir)
		return err
	}
	s.dir = dir
	return nil
}

// The helper takes only the first line, so a reader that lingers after its own read and
// picks up a second copy still sees the right secret.
func (s *Set) feed(path, value string) {
	defer s.wg.Done()
	for {
		f, err := openWhenRead(path, s.done)
		if err != nil {
			return
		}
		f.WriteString(value)
		f.Close()
		select {
		case <-s.done:
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// A FIFO opened O_WRONLY blocks until a reader arrives; O_NONBLOCK turns that wait into
// ENXIO, which lets us poll and still notice that the child has exited.
func openWhenRead(path string, done <-chan struct{}) (*os.File, error) {
	for {
		select {
		case <-done:
			return nil, errors.New("closed")
		default:
		}
		f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, syscall.ENXIO) {
			return nil, err
		}
		select {
		case <-done:
			return nil, errors.New("closed")
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func sanitize(name string) string {
	safe := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			safe = append(safe, r)
		default:
			safe = append(safe, '_')
		}
	}
	return string(safe)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
