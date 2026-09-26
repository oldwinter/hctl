package fsx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/hctl/internal/testutil"
)

func TestSSHViaFakeRunner(t *testing.T) {
	root := t.TempDir()
	s := SSH{Target: "user@box", Run: testutil.SSHLocalRunner(t)}
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

func TestSSHFileOpsThroughRemoteShell(t *testing.T) {
	root := t.TempDir()
	s := SSH{Target: "user@box", Run: testutil.SSHLocalRunner(t)}

	// Write (stdin) and read a path containing a space and an apostrophe.
	p := s.Join(root, "we'ird dir", "file name.txt")
	if err := s.WriteFile(p, []byte("hi there"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hi there" {
		t.Fatalf("%q", got)
	}
	st, err := s.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.IsDir() || st.Size() != 8 {
		t.Fatalf("dir=%v size=%d", st.IsDir(), st.Size())
	}

	// Missing files report os.ErrNotExist.
	missing := s.Join(root, "nope.txt")
	if _, err := s.ReadFile(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("read missing: %v", err)
	}
	if _, err := s.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat missing: %v", err)
	}

	// Rename then remove.
	q := s.Join(root, "ren amed.txt")
	if err := s.Rename(p, q); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Stat(p); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old path after rename: %v", err)
	}
	if err := s.Remove(q); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(q); !os.IsNotExist(err) {
		t.Fatalf("removed path: %v", err)
	}

	// MkdirAll and Stat on a directory.
	d := s.Join(root, "sub dir")
	if err := s.MkdirAll(d, 0o700); err != nil {
		t.Fatal(err)
	}
	st, err = s.Stat(d)
	if err != nil {
		t.Fatal(err)
	}
	if !st.IsDir() {
		t.Fatal("expected dir")
	}
}

func TestSSHLookPathCommandV(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	s := SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			cmd := exec.Command("/bin/sh", "-c", testutil.SSHRemoteCommand(t, args))
			cmd.Env = []string{"PATH=" + dir + ":" + os.Getenv("PATH")}
			return cmd.Output()
		},
	}
	p, err := s.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	if p != bin {
		t.Fatalf("%q", p)
	}
	if _, err := s.LookPath("missing"); err == nil {
		t.Fatal("expected lookup failure")
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

func TestSSHArgsIncludeBatchMode(t *testing.T) {
	s := SSH{Target: "u@h", Identity: "/tmp/id"}
	args := s.args("echo 'a b'")
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
	// The remote command travels as one shell-quoted argument so OpenSSH's
	// space-joining cannot split the script from sh -c.
	if last := args[len(args)-1]; last != `sh -c 'echo '"'"'a b'"'"''` {
		t.Fatalf("%q", last)
	}
}
