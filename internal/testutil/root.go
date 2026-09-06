package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CopyTree copies src directory into a new temp dir and returns that dir.
func CopyTree(t *testing.T, src string) string {
	t.Helper()
	dest, err := copyTree(src, t.TempDir())
	if err != nil {
		panic(err)
	}
	return dest
}

func copyTree(src, dest string) (string, error) {
	src = filepath.Clean(src)
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, src)
		rel = strings.TrimPrefix(rel, string(os.PathSeparator))
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
	if err != nil {
		return "", err
	}
	return dest, nil
}

// RepoRoot walks up from the test working directory to the module root.
func RepoRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	root, err := findRepoRoot(wd)
	if err != nil {
		panic(err)
	}
	return root
}

func findRepoRoot(wd string) (string, error) {
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found from %s", wd)
}

func Testdata(t *testing.T, elem ...string) string {
	t.Helper()
	parts := append([]string{RepoRoot(t), "testdata"}, elem...)
	return filepath.Join(parts...)
}
