package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyTreeRepoRootTestdata(t *testing.T) {
	src := t.TempDir()
	sub := filepath.Join(src, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := CopyTree(t, src)
	got, err := os.ReadFile(filepath.Join(dest, "sub", "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hi" {
		t.Fatalf("%q", got)
	}
	root := RepoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(err)
	}
	td := Testdata(t, "home-a")
	if _, err := os.Stat(td); err != nil {
		t.Fatal(err)
	}
}
