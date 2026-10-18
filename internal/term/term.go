package term

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

var stdinReader = bufio.NewReader(os.Stdin)

var ErrNoTerminal = errors.New("password can be read only from a real terminal; run kdbx-cli from a terminal (or cache the password with the \"cached\" config section) and retry")

func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

func ReadWithPrefill(prompt, prefill string) string {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return readLineWithDefault(prompt, prefill)
	}
	defer term.Restore(fd, oldState)

	fmt.Printf("%s: %s", prompt, prefill)
	return readRawLine(fd, oldState, []rune(prefill))
}

func readLineWithDefault(prompt, prefill string) string {
	fmt.Printf("%s [%s]: ", prompt, prefill)
	line, _ := stdinReader.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return prefill
	}
	return line
}

func readRawLine(fd int, oldState *term.State, buf []rune) string {
	one := make([]byte, 1)
	for {
		n, _ := os.Stdin.Read(one)
		if n == 0 {
			return string(buf)
		}

		b := one[0]
		switch {
		case b == '\r' || b == '\n':
			fmt.Print("\r\n")
			return string(buf)
		case b == 3 || b == 27:
			fmt.Print("\r\n")
			term.Restore(fd, oldState)
			os.Exit(1)
		default:
			buf = editLine(buf, b)
		}
	}
}

func editLine(buf []rune, b byte) []rune {
	switch {
	case b == 127 || b == 8:
		if len(buf) == 0 {
			return buf
		}
		fmt.Print("\b \b")
		return buf[:len(buf)-1]
	case b >= 32:
		os.Stdout.Write([]byte{b})
		return append(buf, rune(b))
	default:
		return buf
	}
}

func hasRealTerminal() bool {
	tty := openTTY()
	if tty == nil {
		return false
	}
	defer tty.Close()
	if !term.IsTerminal(int(tty.Fd())) {
		return false
	}
	for _, f := range []*os.File{os.Stdin, os.Stdout, os.Stderr} {
		if term.IsTerminal(int(f.Fd())) {
			return true
		}
	}
	return false
}

func openTTY() *os.File {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil
	}
	return f
}

func confirmYN(question string) bool {
	if !hasRealTerminal() {
		return false
	}
	tty := openTTY()
	if tty == nil {
		return false
	}
	defer tty.Close()

	fmt.Fprint(tty, question+" [y/N]: ")
	reader := bufio.NewReader(tty)
	line, _ := reader.ReadString('\n')
	return strings.EqualFold(strings.TrimSpace(line), "y")
}

func Confirm(question string, assumeYes bool) bool {
	if assumeYes {
		return true
	}
	return confirmYN(question)
}

func ReadPassword(prompt string) (string, error) {
	if v := os.Getenv("KDBX_CLI_PASSWORD"); v != "" {
		return v, nil
	}
	if !hasRealTerminal() {
		return "", ErrNoTerminal
	}
	tty := openTTY()
	if tty == nil {
		return "", ErrNoTerminal
	}
	defer tty.Close()

	fd := int(tty.Fd())

	fmt.Fprint(tty, prompt)
	pw, err := term.ReadPassword(fd)
	if err != nil {
		fmt.Fprintln(tty)
		return "", err
	}
	fmt.Fprintln(tty)
	return string(pw), nil
}
