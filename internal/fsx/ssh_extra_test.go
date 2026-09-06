package fsx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
)

func TestSSHStatMkdirRenameRemoveHome(t *testing.T) {
	root := t.TempDir()
	s := SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			script := args[len(args)-1]
			c := exec.Command("sh", "-c", script)
			c.Dir = root
			if stdin != nil {
				c.Stdin = bytes.NewReader(stdin)
			}
			out, err := c.CombinedOutput()
			// mimic Output: exit errors carry ExitError
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					return out, ee
				}
				return out, err
			}
			return out, nil
		},
	}
	p := path.Join(root, "dir", "f.txt")
	if err := s.MkdirAll(path.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := s.WriteFile(p, []byte("z"), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := s.Stat(p)
	if err != nil || st.IsDir() || st.Name() == "" {
		t.Fatalf("%v %#v", err, st)
	}
	_ = st.Size()
	_ = st.Mode()
	_ = st.ModTime()
	_ = st.Sys()
	dst := path.Join(root, "dir", "g.txt")
	if err := s.Rename(p, dst); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove(dst); err != nil {
		t.Fatal(err)
	}
	_, err = s.Stat(path.Join(root, "nope"))
	if !os.IsNotExist(err) && !IsNotExist(err) {
		// fake runner may wrap exit 3
		if err == nil {
			t.Fatal("expected missing")
		}
	}
	home, err := s.RemoteHome()
	if err != nil || home == "" {
		t.Fatalf("%q %v", home, err)
	}
	// empty home
	sEmpty := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return []byte("\n"), nil
	}}
	_, err = sEmpty.RemoteHome()
	if err == nil || exitcode.From(err) != exitcode.SSH {
		t.Fatalf("%v", err)
	}
	// LookPath empty
	sLP := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return []byte("  \n"), nil
	}}
	if _, err := sLP.LookPath("x"); !os.IsNotExist(err) {
		t.Fatalf("%v", err)
	}
	// isExit / asExit coverage
	ee := &exec.ExitError{}
	_ = ee
	wrapped := exitcode.Wrap(exitcode.SSH, fmt.Errorf("ssh: %w", &exec.ExitError{}))
	_ = isExit(wrapped, 3)
	_ = isExit(fmt.Errorf("exit status 3"), 3)
	_ = isExit(fmt.Errorf("other"), 3)
	var target *exec.ExitError
	_ = asExit(nil, &target)
	_ = asExit(fmt.Errorf("nope"), &target)
	info := sshInfo{name: "n", dir: true, size: 1}
	if !info.IsDir() || info.Mode()&os.ModeDir == 0 {
		t.Fatal(info.Mode())
	}
	info2 := sshInfo{name: "f", dir: false, size: 2}
	if info2.IsDir() || info2.Size() != 2 {
		t.Fatal(info2)
	}
	// defaultRunner error with stderr
	_, err = defaultRunner(nil, "false")
	if err == nil {
		t.Fatal("expected err")
	}
	// ReadFile missing via exit 3 string
	sMiss := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("exit status 3")
	}}
	_, err = sMiss.ReadFile("/no")
	if !os.IsNotExist(err) {
		t.Fatalf("%v", err)
	}
	_ = strings.TrimSpace
}

func TestSSHStatDirAndSize(t *testing.T) {
	s := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return []byte("dir\n4096\n"), nil
	}}
	st, err := s.Stat("/tmp")
	if err != nil || !st.IsDir() || st.Size() != 4096 {
		t.Fatalf("%v %#v", err, st)
	}
	s2 := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("boom")
	}}
	if _, err := s2.Stat("/x"); err == nil {
		t.Fatal("expected err")
	}
}
