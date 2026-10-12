package secretpipe

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestFileIsReadableRepeatedly(t *testing.T) {
	s := NewSet()
	defer s.Close()

	f, err := s.File("repo pw", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cmd := exec.Command("sh", "-c", "cat /dev/fd/3; cat /dev/fd/3")
	cmd.ExtraFiles = []*os.File{f}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "s3crets3cret" {
		t.Errorf("got %q, want %q", string(out), "s3crets3cret")
	}
}

func TestFileIsOwnerOnly(t *testing.T) {
	s := NewSet()
	defer s.Close()

	f, err := s.File("pw", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	cmd := exec.Command("stat", "-L", "-c", "%a", "/dev/fd/3")
	cmd.ExtraFiles = []*os.File{f}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "600" {
		t.Errorf("mode of /dev/fd/3: got %s, want 600", got)
	}
}

func TestAskpassScript(t *testing.T) {
	s := NewSet()
	defer s.Close()

	script, err := s.Askpass("pw", "s3cret", 3)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 3 {
		out, err := exec.Command(script, "Password:").Output()
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if string(out) != "s3cret\n" {
			t.Errorf("call %d: got %q", i, string(out))
		}
	}
}

func TestCloseWithoutReader(t *testing.T) {
	s := NewSet()
	if _, err := s.Askpass("pw", "s3cret", 0); err != nil {
		t.Fatal(err)
	}
	dir := s.Dir()

	done := make(chan struct{})
	go func() {
		s.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close blocked with no reader attached")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("directory %s was not removed", dir)
	}
}

func TestAskpassDirIsPrivate(t *testing.T) {
	s := NewSet()
	defer s.Close()

	if _, err := s.Askpass("pw", "v", 0); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(s.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0700 {
		t.Errorf("mode: got %o, want 700", fi.Mode().Perm())
	}
}

func TestAskpassServesOnceByDefault(t *testing.T) {
	s := NewSet()
	defer s.Close()

	script, err := s.Askpass("pw", "s3cret", 0)
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(script, "Password:").Output()
	if err != nil || string(out) != "s3cret\n" {
		t.Fatalf("first call: %q %v", out, err)
	}
	time.Sleep(50 * time.Millisecond)
	out, _ = exec.Command(script, "Password:").Output()
	if string(out) != "" {
		t.Errorf("second call should get nothing, got %q", out)
	}
}

func TestAskpassRefusesUsernameAndHostKeyPrompts(t *testing.T) {
	s := NewSet()
	defer s.Close()

	script, err := s.Askpass("pw", "s3cret", 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, prompt := range []string{
		"Username for 'https://example.com': ",
		"Are you sure you want to continue connecting (yes/no/[fingerprint])? ",
	} {
		out, err := exec.Command(script, prompt).Output()
		if err == nil || len(out) != 0 {
			t.Errorf("%q: should be refused, got %q err=%v", prompt, out, err)
		}
	}
	out, err := exec.Command(script, "Password for 'https://alice@example.com': ").Output()
	if err != nil || string(out) != "s3cret\n" {
		t.Errorf("password prompt after refusals: %q %v", out, err)
	}
}
