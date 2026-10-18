package deliver

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdinPayload(t *testing.T) {
	got, err := stdinPayload([]string{"pw", "pw"}, map[string]string{"pw": "s3cret"})
	require.NoError(t, err)
	assert.Equal(t, "s3cret\ns3cret\n", got)
}

func TestStdinPayloadRejectsMultilineSecret(t *testing.T) {
	_, err := stdinPayload([]string{"pw"}, map[string]string{"pw": "a\nb"})
	assert.Error(t, err)
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(data)
}

func TestStdinWithoutSecretsPassesParent(t *testing.T) {
	parent := strings.NewReader("typed")

	got, err := Stdin(nil, false, nil, parent)

	require.NoError(t, err)
	assert.Same(t, parent, got)
}

func TestStdinClosedAfterSecrets(t *testing.T) {
	got, err := Stdin([]string{"pw"}, false, map[string]string{"pw": "s"}, strings.NewReader("typed"))

	require.NoError(t, err)
	assert.Equal(t, "s\n", readAll(t, got))
}

func TestStdinKeepOpenAppendsParent(t *testing.T) {
	got, err := Stdin([]string{"pw"}, true, map[string]string{"pw": "s"}, strings.NewReader("typed"))

	require.NoError(t, err)
	assert.Equal(t, "s\ntyped", readAll(t, got))
}

func TestFilesPathsNumberFromFirstExtraFD(t *testing.T) {
	files := Files{{name: "a"}, {name: "b"}}

	assert.Equal(t, map[string]string{"a": "/dev/fd/3", "b": "/dev/fd/4"}, files.Paths())
}

func TestSubstitutePlaceholders(t *testing.T) {
	args := []string{"restic", "--password-file", "{{repo pw}}", "--other=x{{repo pw}}x"}

	got, err := SubstitutePlaceholders(args, map[string]string{"repo pw": "/run/fifo"})

	t.Run("succeeds", func(t *testing.T) {
		require.NoError(t, err)
	})
	t.Run("substitutes whole and embedded placeholders", func(t *testing.T) {
		assert.Equal(t, []string{"restic", "--password-file", "/run/fifo", "--other=x/run/fifox"}, got)
	})
	t.Run("does not mutate input", func(t *testing.T) {
		assert.Equal(t, "{{repo pw}}", args[2])
	})
}

func TestSubstitutePlaceholdersMissingPlaceholder(t *testing.T) {
	_, err := SubstitutePlaceholders([]string{"restic"}, map[string]string{"pw": "/run/fifo"})
	assert.Error(t, err)
}

func TestWithAskpass(t *testing.T) {
	env := WithAskpass([]string{"PATH=/bin", "SSH_ASKPASS=/old", "HOME=/h"}, "/tmp/askpass.sh")

	t.Run("replaces existing askpass", func(t *testing.T) {
		assert.NotContains(t, env, "SSH_ASKPASS=/old")
	})
	t.Run("keeps unrelated variables", func(t *testing.T) {
		assert.Subset(t, env, []string{"PATH=/bin", "HOME=/h"})
	})
	t.Run("points helpers at script", func(t *testing.T) {
		assert.Subset(t, env, []string{
			"SSH_ASKPASS=/tmp/askpass.sh", "SUDO_ASKPASS=/tmp/askpass.sh",
			"GIT_ASKPASS=/tmp/askpass.sh", "RESTIC_PASSWORD_COMMAND=/tmp/askpass.sh",
			"BORG_PASSCOMMAND=/tmp/askpass.sh",
		})
	})
	t.Run("forces askpass and disables prompts", func(t *testing.T) {
		assert.Subset(t, env, []string{"SSH_ASKPASS_REQUIRE=force", "GIT_TERMINAL_PROMPT=0"})
	})
}

func TestInheritedEnvDropsPassword(t *testing.T) {
	got := InheritedEnv([]string{"A=1", "KDBX_CLI_PASSWORD=secret", "KDBX_CLI_PASSWORD_X=keep", "B=2"})
	assert.Equal(t, []string{"A=1", "KDBX_CLI_PASSWORD_X=keep", "B=2"}, got)
}
