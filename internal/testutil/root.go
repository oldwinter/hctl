package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// RepoRoot walks up from the test working directory to the module root.
func RepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("go.mod not found from " + wd)
	return ""
}

func Testdata(t *testing.T, elem ...string) string {
	t.Helper()
	parts := append([]string{RepoRoot(t), "testdata"}, elem...)
	return filepath.Join(parts...)
}
