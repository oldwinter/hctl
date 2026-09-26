package testutil

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// SSHRemoteCommand reconstructs the command string ssh(1) sends to the
// server from the argv a Runner receives: the destination is the first
// non-option argument, a "--" directly after it is consumed by ssh's own
// option parsing, and the remaining arguments are appended to the command
// separated by spaces. Tests assert on this string (or run it) instead of
// trusting the final argv slot.
func SSHRemoteCommand(t *testing.T, args []string) string {
	t.Helper()
	i := 0
	for i < len(args) && strings.HasPrefix(args[i], "-") {
		switch args[i] {
		case "-o", "-i":
			i += 2
		default:
			t.Fatalf("unexpected ssh option %q in %v", args[i], args)
		}
	}
	if i >= len(args) {
		t.Fatalf("ssh args %v: no destination", args)
	}
	i++ // destination
	rest := args[i:]
	if len(rest) > 0 && rest[0] == "--" {
		rest = rest[1:]
	}
	if len(rest) == 0 {
		t.Fatalf("ssh args %v: no remote command", args)
	}
	return strings.Join(rest, " ")
}

// SSHShellRunner returns an fsx.SSH Run implementation that executes the
// reconstructed remote command through the local shell, the way sshd runs it
// remotely via the login shell's -c. It crosses the same quoting boundary a
// real SSH round trip does; no SSH server or credentials are needed.
func SSHShellRunner(t *testing.T) func(stdin []byte, name string, args ...string) ([]byte, error) {
	return func(stdin []byte, name string, args ...string) ([]byte, error) {
		t.Helper()
		if name != "ssh" {
			t.Fatalf("name=%q", name)
		}
		cmd := exec.Command("sh", "-c", SSHRemoteCommand(t, args))
		if stdin != nil {
			cmd.Stdin = bytes.NewReader(stdin)
		}
		return cmd.Output()
	}
}
