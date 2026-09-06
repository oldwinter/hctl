package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyTreeAndFindRepoRootErrors(t *testing.T) {
	if runtime.GOOS != "windows" {
		src := t.TempDir()
		p := filepath.Join(src, "x")
		os.WriteFile(p, []byte("x"), 0o644)
		os.Chmod(p, 0o000)
		defer os.Chmod(p, 0o644)
		if _, err := copyTree(src, t.TempDir()); err == nil {
			t.Fatal("expected read err")
		}
		if _, err := copyTree(filepath.Join(src, "missing"), t.TempDir()); err == nil {
			t.Fatal("expected walk err")
		}
	}
	if _, err := findRepoRoot(filepath.Join(t.TempDir(), "nowhere")); err == nil {
		t.Fatal("expected no go.mod")
	}
	func() {
		defer func() { _ = recover() }()
		CopyTree(t, filepath.Join(t.TempDir(), "missing-tree"))
		t.Fatal("expected panic")
	}()
	func() {
		defer func() { _ = recover() }()
		// force findRepoRoot fail via temporary chdir? RepoRoot uses Getwd
		// call findRepoRoot path through panic in RepoRoot by swapping — just panic path:
		oldwd, _ := os.Getwd()
		os.Chdir(t.TempDir())
		defer os.Chdir(oldwd)
		RepoRoot(t)
		t.Fatal("expected panic")
	}()
}
