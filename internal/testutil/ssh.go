package testutil

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// SSHRemoteCommand extracts the remote command line from an fsx.SSH
// invocation the way OpenSSH builds it: local option arguments and the
// destination are skipped, a leading "--" after the destination is consumed
// by ssh's own option parsing, and every remaining argument is joined with
// single spaces into the string the remote shell receives.
func SSHRemoteCommand(t *testing.T, args []string) string {
	t.Helper()
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "-o" || a == "-i" {
			i += 2 // option with a separate value
			continue
		}
		if strings.HasPrefix(a, "-") {
			i++
			continue
		}
		break
	}
	if i < len(args) {
		i++ // destination
	}
	if i < len(args) && args[i] == "--" {
		i++
	}
	if i >= len(args) {
		t.Fatalf("no remote command in args %v", args)
	}
	return strings.Join(args[i:], " ")
}

// SSHLocalRunner emulates a remote shell for fsx.SSH tests: the remote
// command OpenSSH would send is run through the local /bin/sh -c. The
// signature matches fsx.Runner without importing fsx.
func SSHLocalRunner(t *testing.T) func(stdin []byte, name string, args ...string) ([]byte, error) {
	t.Helper()
	return func(stdin []byte, name string, args ...string) ([]byte, error) {
		if name != "ssh" {
			t.Fatalf("expected ssh, got %q", name)
		}
		cmd := exec.Command("/bin/sh", "-c", SSHRemoteCommand(t, args))
		if stdin != nil {
			cmd.Stdin = bytes.NewReader(stdin)
		}
		return cmd.Output()
	}
}
