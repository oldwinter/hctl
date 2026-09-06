package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// CopyTree copies src directory into a new temp dir and returns that dir.
func CopyTree(t *testing.T, src string) string {
	t.Helper()
	dest := t.TempDir()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
	if err != nil {
		t.Fatal(err)
	}
	return dest
}

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
