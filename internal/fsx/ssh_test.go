package fsx

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// localRunner executes the remote sh -c script locally under root.
func localRunner(t *testing.T, root string) Runner {
	t.Helper()
	return func(stdin []byte, name string, args ...string) ([]byte, error) {
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
	}
}

func TestSSHViaFakeRunner(t *testing.T) {
	root := t.TempDir()
	s := SSH{
		Target: "user@box",
		Run:    localRunner(t, root),
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

func TestSSHWriteNewFileRejectsExisting(t *testing.T) {
	root := t.TempDir()
	s := SSH{Target: "user@box", Run: localRunner(t, root)}
	existing := filepath.Join(root, "exists.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteNewFile(existing, []byte("evil"), 0o600); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("existing file: err=%v want ErrExist", err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(existing, link); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteNewFile(link, []byte("evil"), 0o600); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("symlink: err=%v want ErrExist", err)
	}
	dangling := filepath.Join(root, "dangling")
	if err := os.Symlink(filepath.Join(root, "nowhere"), dangling); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteNewFile(dangling, []byte("evil"), 0o600); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("dangling symlink: err=%v want ErrExist", err)
	}
	if data, err := os.ReadFile(existing); err != nil || string(data) != "keep" {
		t.Fatalf("target=%q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, "nowhere")); !os.IsNotExist(err) {
		t.Fatalf("dangling target created: %v", err)
	}
	fresh := filepath.Join(root, "fresh.txt")
	if err := s.WriteNewFile(fresh, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(fresh); err != nil || string(data) != "new" {
		t.Fatalf("fresh=%q err=%v", data, err)
	}
}

// sshPublishRe locates the hard-link publish step inside WriteNewFile's
// remote script; group 1 is the reserved temp path.
var sshPublishRe = regexp.MustCompile(`ln '[^']*' '([^']*)'`)

// plantingRunner executes the remote script locally like localRunner, but
// injects plant(tmp) between the mkdir reservation and the ln publish of the
// first WriteNewFile attempt — a racer that lands after the exclusive check.
func plantingRunner(t *testing.T, root string, plant func(tmp string) string) Runner {
	t.Helper()
	injected := false
	return func(stdin []byte, name string, args ...string) ([]byte, error) {
		if name != "ssh" {
			t.Fatalf("name=%s", name)
		}
		script := args[len(args)-1]
		if !injected && strings.Contains(script, "&& ln ") {
			m := sshPublishRe.FindStringSubmatch(script)
			if m == nil {
				t.Fatalf("no publish step in %q", script)
			}
			script = strings.Replace(script, "&& ln ", "&& "+plant(m[1])+" && ln ", 1)
			injected = true
		}
		c := exec.Command("sh", "-c", script)
		c.Dir = root
		if stdin != nil {
			c.Stdin = bytes.NewReader(stdin)
		}
		return c.Output()
	}
}

func TestSSHAtomicWriteRejectsRacingFIFOSymlink(t *testing.T) {
	root := t.TempDir()
	fifo := filepath.Join(root, "planted.fifo")
	if out, err := exec.Command("mkfifo", fifo).CombinedOutput(); err != nil {
		t.Fatalf("mkfifo: %v %s", err, out)
	}
	// O_RDWR opens without a peer and never reports EOF, so any read means
	// the payload traversed the planted symlink into the FIFO.
	pipe, err := os.OpenFile(fifo, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pipe.Close() }()
	received := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 4096)
		n, err := pipe.Read(buf)
		if err != nil {
			n = 0
		}
		received <- buf[:n]
	}()

	var planted string
	s := SSH{Target: "user@box", Run: plantingRunner(t, root, func(tmp string) string {
		planted = tmp
		return fmt.Sprintf("ln -s %s %s", shq(fifo), shq(tmp))
	})}
	dest := s.Join(root, "config.json")
	if err := AtomicWrite(s, dest, []byte("new-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if planted == "" {
		t.Fatal("plant never ran")
	}
	st, err := os.Lstat(planted)
	if err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("planted tmp=%v err=%v", st, err)
	}
	select {
	case b := <-received:
		t.Fatalf("payload reached fifo: %q", b)
	case <-time.After(300 * time.Millisecond):
	}
	st, err = os.Lstat(dest)
	if err != nil || !st.Mode().IsRegular() {
		t.Fatalf("dest=%v err=%v", st, err)
	}
	if data, err := os.ReadFile(dest); err != nil || string(data) != "new-config" {
		t.Fatalf("dest=%q err=%v", data, err)
	}
}

func TestSSHAtomicWritePreservesRacingRegularFile(t *testing.T) {
	root := t.TempDir()
	var planted string
	s := SSH{Target: "user@box", Run: plantingRunner(t, root, func(tmp string) string {
		planted = tmp
		return fmt.Sprintf("printf %%s %s > %s", shq("other-writer"), shq(tmp))
	})}
	dest := s.Join(root, "config.json")
	if err := AtomicWrite(s, dest, []byte("new-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if planted == "" {
		t.Fatal("plant never ran")
	}
	if data, err := os.ReadFile(planted); err != nil || string(data) != "other-writer" {
		t.Fatalf("competing temp overwritten: %q err=%v", data, err)
	}
	if data, err := os.ReadFile(dest); err != nil || string(data) != "new-config" {
		t.Fatalf("dest=%q err=%v", data, err)
	}
}

func TestSSHAtomicWriteMustNotFollowStaleTempSymlink(t *testing.T) {
	root := t.TempDir()
	s := SSH{Target: "user@box", Run: localRunner(t, root)}
	dest := s.Join(root, "config.json")
	unrelated := filepath.Join(root, "unrelated.txt")
	if err := os.WriteFile(unrelated, []byte("keep-me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(unrelated, dest+".harnessctl-tmp"); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(s, dest, []byte("new-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(unrelated); err != nil || string(data) != "keep-me" {
		t.Fatalf("unrelated=%q err=%v", data, err)
	}
	st, err := os.Lstat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t.Fatal("destination became a symlink")
	}
	if data, err := os.ReadFile(dest); err != nil || string(data) != "new-config" {
		t.Fatalf("dest=%q err=%v", data, err)
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
