package fsx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
)

func TestFSXRemainingGaps(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "f")
	if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BackupLocal(filepath.Join(f, "x"), "h", []byte("data")); err == nil {
		t.Fatal("backup mkdir")
	}
	if _, err := defaultRunner([]byte("hi"), "cat"); err != nil {
		t.Fatal(err)
	}
	if _, err := defaultRunner(nil, "sh", "-c", "exit 2"); err == nil {
		t.Fatal("exit")
	}
	s := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return nil, errors.New("no")
	}}
	if _, err := s.LookPath("x"); err == nil {
		t.Fatal("lp")
	}
	s2 := SSH{Target: "invalid-host-for-hctl-test"}
	if _, err := s2.run(nil, "true"); err == nil {
		t.Fatal("ssh")
	}
	cmd := exec.Command("sh", "-c", "exit 3")
	err := cmd.Run()
	wrapped := exitcode.Wrap(exitcode.SSH, fmt.Errorf("x: %w", err))
	if !isExit(wrapped, 3) {
		t.Fatal("isExit wrap")
	}
	_ = bytes.MinRead
}

func TestBackupLocalWriteFail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(dir, 0o755)
	if _, err := BackupLocal(dir, "h", []byte("data")); err == nil {
		t.Fatal("expected write fail")
	}
}
