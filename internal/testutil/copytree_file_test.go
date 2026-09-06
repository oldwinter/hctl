package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTreeMkdirAllOnFileParent(t *testing.T) {
	srcFile := filepath.Join(t.TempDir(), "src")
	os.WriteFile(srcFile, []byte("x"), 0o644)
	blocker := filepath.Join(t.TempDir(), "file")
	os.WriteFile(blocker, []byte("x"), 0o644)
	if _, err := copyTree(srcFile, filepath.Join(blocker, "child")); err == nil {
		t.Fatal("expected")
	}
}
