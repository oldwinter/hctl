package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTreeMkdirFail(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "f"), []byte("x"), 0o644)
	destParent := t.TempDir()
	blocking := filepath.Join(destParent, "block")
	os.WriteFile(blocking, []byte("x"), 0o644)
	// copyTree into a path where a file blocks a directory create
	if _, err := copyTree(src, blocking); err == nil {
		t.Fatal("expected")
	}
	func() {
		defer func() { _ = recover() }()
		wd, _ := os.Getwd()
		_ = wd
		// panic Getwd: can't; panic findRepoRoot already tested
	}()
}
