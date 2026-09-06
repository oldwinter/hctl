package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTreeDestFileBlocksDir(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "sub", "f"), []byte("x"), 0o644)
	dest := t.TempDir()
	// create a FILE named "sub" so MkdirAll for nested file parent somehow —
	// actually Walk creates sub as dir first. Force WriteFile into file-as-dir:
	os.WriteFile(filepath.Join(dest, "sub"), []byte("x"), 0o644)
	if _, err := copyTree(src, dest); err == nil {
		t.Fatal("expected")
	}
	if _, err := findRepoRoot("/"); err == nil {
		t.Log("go.mod at /?")
	}
}
