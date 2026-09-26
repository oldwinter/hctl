package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/oldwinter/hctl/internal/testutil"
)

func TestSSHViaFakeRunner(t *testing.T) {
	root := t.TempDir()
	s := SSH{
		Target: "user@box",
		Run:    testutil.SSHShellRunner(t),
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

// TestSSHRemoteShellRoundTrip exercises read, write-with-stdin, stat, rename,
// remove, and RemoteHome through the joined remote command, including paths
// that need quoting (space and apostrophe). The runner joins argv exactly
// like ssh(1) does, so a quoting bug fails here without a server.
func TestSSHRemoteShellRoundTrip(t *testing.T) {
	root := t.TempDir()
	s := SSH{
		Target: "user@box",
		Run:    testutil.SSHShellRunner(t),
	}
	p := s.Join(root, "sub dir", "it's a file.txt")
	if err := s.WriteFile(p, []byte("hi there\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hi there\n" {
		t.Fatalf("%q", got)
	}
	st, err := s.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.IsDir() || st.Size() != 9 {
		t.Fatalf("stat: dir=%v size=%d", st.IsDir(), st.Size())
	}
	renamed := s.Join(root, "renamed.txt")
	if err := s.Rename(p, renamed); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadFile(p); !os.IsNotExist(err) {
		t.Fatalf("renamed-away source: %v", err)
	}
	if _, err := s.Stat(s.Join(root, "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing stat: %v", err)
	}
	if _, err := s.ReadFile(s.Join(root, "missing")); !os.IsNotExist(err) {
		t.Fatalf("missing read: %v", err)
	}
	if err := s.Remove(renamed); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Stat(renamed); !os.IsNotExist(err) {
		t.Fatalf("removed stat: %v", err)
	}
	if os.Getenv("HOME") != "" {
		home, err := s.RemoteHome()
		if err != nil || home != os.Getenv("HOME") {
			t.Fatalf("home=%q err=%v", home, err)
		}
	}
}

func TestSSHLookPathCommandV(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fixture-codex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	s := SSH{
		Target: "user@box",
		Run:    testutil.SSHShellRunner(t),
	}
	p, err := s.LookPath("fixture-codex")
	if err != nil {
		t.Fatal(err)
	}
	if p != bin {
		t.Fatalf("%q", p)
	}
	if _, err := s.LookPath("fixture-missing"); err == nil {
		t.Fatal("expected missing binary error")
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

// The remote script must be a single shell-quoted argument: OpenSSH appends
// command arguments to the remote command separated by spaces, so an
// unquoted script would be reparsed as part of the outer sh -c invocation.
func TestSSHArgsQuoteRemoteScript(t *testing.T) {
	s := SSH{Target: "u@h"}
	args := s.args("echo a; echo $HOME")
	last := args[len(args)-1]
	if last != "'echo a; echo $HOME'" {
		t.Fatalf("%v", args)
	}
	args = s.args("echo it's")
	if args[len(args)-1] != `'echo it'"'"'s'` {
		t.Fatalf("%v", args)
	}
}
