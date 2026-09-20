package fsx

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/hctl/internal/exitcode"
)

func TestSSHViaFakeRunner(t *testing.T) {
	root := t.TempDir()
	s := SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			if name != "ssh" {
				t.Fatalf("name=%s", name)
			}
			script := args[len(args)-1]
			c := exec.Command("sh", "-c", script)
			c.Dir = root
			if stdin != nil {
				c.Stdin = bytes.NewReader(stdin)
			}
			out, err := c.Output()
			return out, err
		},
	}
	p := s.Join(root, "file.txt")
	if err := s.WriteFile(p, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("%q", got)
	}
}

func TestSSHLookPathCommandV(t *testing.T) {
	s := SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			script := args[len(args)-1]
			if script != "command -v 'codex'" {
				t.Fatalf("script=%q", script)
			}
			return []byte("/usr/bin/codex\n"), nil
		},
	}
	p, err := s.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	if p != "/usr/bin/codex" {
		t.Fatalf("%q", p)
	}
	if _, err := s.LookPath("codex; rm -rf /"); err == nil {
		t.Fatal("expected invalid name rejection")
	}
}

func TestDefaultRunnerTimeout(t *testing.T) {
	old := runTimeout
	runTimeout = 200 * time.Millisecond
	defer func() { runTimeout = old }()
	_, err := defaultRunner(nil, "sleep", "5")
	if err == nil {
		t.Fatal("expected timeout")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("%v", err)
	}
}

func TestSSHReadFileMissingMapsNotExist(t *testing.T) {
	s := SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			cmd := exec.Command("sh", "-c", "exit 3")
			return nil, cmd.Run()
		},
	}
	_, err := s.ReadFile("/nope")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err=%v", err)
	}
}

func TestSSHRemoteHomeEmpty(t *testing.T) {
	s := SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			return []byte("  \n"), nil
		},
	}
	_, err := s.RemoteHome()
	if exitcode.From(err) != exitcode.SSH {
		t.Fatalf("err=%v code=%d", err, exitcode.From(err))
	}
}

func TestSSHArgsIncludeBatchMode(t *testing.T) {
	s := SSH{Target: "u@h", Identity: "/tmp/id"}
	args := s.args("true")
	ok := false
	for i, a := range args {
		if a == "-o" && i+1 < len(args) && args[i+1] == "BatchMode=yes" {
			ok = true
		}
		if a == "-i" && i+1 < len(args) && args[i+1] == "/tmp/id" {
			ok = ok && true
		}
	}
	if !ok {
		t.Fatalf("%v", args)
	}
}
