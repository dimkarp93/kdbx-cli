package deliver

import (
	"fmt"
	"io"
	"strings"
)

func Stdin(names []string, keepOpen bool, values map[string]string, parent io.Reader) (io.Reader, error) {
	if len(names) == 0 {
		return parent, nil
	}

	payload, err := stdinPayload(names, values)
	if err != nil {
		return nil, err
	}

	if keepOpen {
		return io.MultiReader(strings.NewReader(payload), parent), nil
	}
	return strings.NewReader(payload), nil
}

func stdinPayload(names []string, values map[string]string) (string, error) {
	var b strings.Builder
	for _, name := range names {
		v := values[name]
		if strings.ContainsAny(v, "\r\n") {
			return "", fmt.Errorf("secret %q contains a newline and cannot be sent over stdin", name)
		}
		b.WriteString(v)
		b.WriteByte('\n')
	}
	return b.String(), nil
}
