package deliver

const passwordEnv = "KDBX_CLI_PASSWORD"

var AskpassEnv = []string{"SSH_ASKPASS", "SUDO_ASKPASS", "GIT_ASKPASS", "RESTIC_PASSWORD_COMMAND", "BORG_PASSCOMMAND"}

func envKey(kv string) string {
	panic("not implemented")
}
func InheritedEnv(environ []string) []string {
	panic("not implemented")
}
func SecretEnv(secrets, values map[string]string) []string {
	panic("not implemented")
}
func WithAskpass(env []string, script string) []string {
	panic("not implemented")
}
func askpassOverrides(script string) map[string]string {
	panic("not implemented")
}
