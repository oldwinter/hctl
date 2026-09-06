package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTreeFileAndRepoRoot(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "f"), []byte("x"), 0o644)
	_ = CopyTree(t, src)
	// Testdata nested
	_ = Testdata(t)
}
