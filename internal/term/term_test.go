package term

import (
	"errors"
	"testing"
)

func TestReadPasswordFromEnv(t *testing.T) {
	t.Setenv("KDBX_CLI_PASSWORD", "secret")
	pw, err := ReadPassword("prompt: ")
	if err != nil || pw != "secret" {
		t.Errorf("pw=%q err=%v", pw, err)
	}
}

func TestReadPasswordWithoutTerminal(t *testing.T) {
	t.Setenv("KDBX_CLI_PASSWORD", "")
	if hasRealTerminal() {
		t.Skip("test runs with a real terminal")
	}
	pw, err := ReadPassword("prompt: ")
	if !errors.Is(err, ErrNoTerminal) || pw != "" {
		t.Errorf("pw=%q err=%v", pw, err)
	}
}

func TestConfirmWithoutTerminal(t *testing.T) {
	if hasRealTerminal() {
		t.Skip("test runs with a real terminal")
	}
	if Confirm("q?", false) {
		t.Error("Confirm must decline without a terminal")
	}
	if !Confirm("q?", true) {
		t.Error("assumeYes must still confirm")
	}
}
