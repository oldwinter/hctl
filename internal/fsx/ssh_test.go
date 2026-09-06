package fsx

import (
	"bytes"
	"os/exec"
	"testing"
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
