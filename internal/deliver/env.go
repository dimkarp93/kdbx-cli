package deliver

import (
	"slices"
	"strings"

	"github.com/dimkarp93/kdbx-cli/internal/utils"
)

const passwordEnv = "KDBX_CLI_PASSWORD"

var AskpassEnv = []string{
	"SSH_ASKPASS",
	"SUDO_ASKPASS",
	"GIT_ASKPASS",
	"RESTIC_PASSWORD_COMMAND",
	"BORG_PASSCOMMAND",
}

func envKey(kv string) string {
	k, _, _ := strings.Cut(kv, "=")
	return k
}

func InheritedEnv(environ []string) []string {
	return utils.Filter(environ, func(kv string) bool { return envKey(kv) != passwordEnv })
}

func SecretEnv(secrets, values map[string]string) []string {
	return utils.Map(utils.SortedKeys(secrets), func(env string) string {
		return env + "=" + values[secrets[env]]
	})
}

func WithAskpass(env []string, script string) []string {
	overrides := askpassOverrides(script)
	kept := utils.Filter(env, func(kv string) bool {
		_, overridden := overrides[envKey(kv)]
		return !overridden
	})
	added := utils.Map(utils.SortedKeys(overrides), func(k string) string { return k + "=" + overrides[k] })

	return slices.Concat(kept, added)
}

func askpassOverrides(script string) map[string]string {
	overrides := map[string]string{
		"SSH_ASKPASS_REQUIRE": "force",
		"GIT_TERMINAL_PROMPT": "0",
	}
	for _, k := range AskpassEnv {
		overrides[k] = script
	}
	return overrides
}
