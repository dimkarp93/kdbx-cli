package secretpipe

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scriptRun struct {
	out []byte
	err error
}

func newTestSet(t *testing.T) *Set {
	t.Helper()
	s := NewSet()
	t.Cleanup(s.Close)
	return s
}

func secretFile(t *testing.T, s *Set, value string) *os.File {
	t.Helper()

	f, err := s.File("pw", value)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	return f
}

func askpassScript(t *testing.T, s *Set, reads int) string {
	t.Helper()
	script, err := s.Askpass("pw", "s3cret", reads)
	require.NoError(t, err)
	return script
}

func withFD3(f *os.File, name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.ExtraFiles = []*os.File{f}
	return cmd
}

func runScript(script, prompt string) scriptRun {
	out, err := exec.Command(script, prompt).Output()
	return scriptRun{out: out, err: err}
}

func closeWithin(s *Set, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		s.Close()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestFileIsReadableRepeatedly(t *testing.T) {
	f := secretFile(t, newTestSet(t), "s3cret")

	out, err := withFD3(f, "sh", "-c", "cat /dev/fd/3; cat /dev/fd/3").Output()

	require.NoError(t, err)
	assert.Equal(t, "s3crets3cret", string(out))
}

func TestFileIsOwnerOnly(t *testing.T) {
	f := secretFile(t, newTestSet(t), "s3cret")

	out, err := withFD3(f, "stat", "-L", "-c", "%a", "/dev/fd/3").Output()

	require.NoError(t, err)
	assert.Equal(t, "600", strings.TrimSpace(string(out)))
}

func TestFileIsSealed(t *testing.T) {
	f := secretFile(t, newTestSet(t), "s3cret")

	tamperErr := withFD3(f, "sh", "-c", "echo tampered > /dev/fd/3").Run()
	out, readErr := withFD3(f, "cat", "/dev/fd/3").Output()

	t.Run("rewrite refused", func(t *testing.T) {
		assert.Error(t, tamperErr)
	})
	t.Run("content intact", func(t *testing.T) {
		require.NoError(t, readErr)
		assert.Equal(t, "s3cret", string(out))
	})
}

func TestAskpassServesConfiguredReads(t *testing.T) {
	script := askpassScript(t, newTestSet(t), 3)

	for i := range 3 {
		run := runScript(script, "Password:")
		t.Run(fmt.Sprintf("call %d", i+1), func(t *testing.T) {
			require.NoError(t, run.err)
			assert.Equal(t, "s3cret\n", string(run.out))
		})
	}
}

func TestAskpassServesOnceByDefault(t *testing.T) {
	script := askpassScript(t, newTestSet(t), 0)

	first := runScript(script, "Password:")
	time.Sleep(50 * time.Millisecond)
	second := runScript(script, "Password:")

	t.Run("first call gets secret", func(t *testing.T) {
		require.NoError(t, first.err)
		assert.Equal(t, "s3cret\n", string(first.out))
	})
	t.Run("second call gets nothing", func(t *testing.T) {
		assert.Empty(t, second.out)
	})
}

func TestAskpassRefusesUsernameAndHostKeyPrompts(t *testing.T) {
	script := askpassScript(t, newTestSet(t), 1)

	refused := map[string]scriptRun{}
	for _, prompt := range []string{
		"Username for 'https://example.com': ",
		"Are you sure you want to continue connecting (yes/no/[fingerprint])? ",
	} {
		refused[prompt] = runScript(script, prompt)
	}
	password := runScript(script, "Password for 'https://alice@example.com': ")

	for prompt, run := range refused {
		t.Run("refuses "+prompt, func(t *testing.T) {
			assert.Error(t, run.err)
			assert.Empty(t, run.out)
		})
	}
	t.Run("password prompt still served", func(t *testing.T) {
		require.NoError(t, password.err)
		assert.Equal(t, "s3cret\n", string(password.out))
	})
}

func TestAskpassDirIsPrivate(t *testing.T) {
	s := newTestSet(t)
	askpassScript(t, s, 0)

	fi, err := os.Stat(s.Dir())

	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), fi.Mode().Perm())
}

func TestCloseWithoutReader(t *testing.T) {
	s := NewSet()
	askpassScript(t, s, 0)
	dir := s.Dir()

	closed := closeWithin(s, 5*time.Second)

	t.Run("does not block", func(t *testing.T) {
		require.True(t, closed)
	})
	t.Run("removes directory", func(t *testing.T) {
		assert.NoDirExists(t, dir)
	})
}
