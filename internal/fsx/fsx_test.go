package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFSHelpers(t *testing.T) {
	var l Local
	dir := t.TempDir()
	p := filepath.Join(dir, "a.txt")
	if err := l.WriteFile(p, []byte("hi"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := l.ReadFile(p)
	if err != nil || string(got) != "hi" {
		t.Fatalf("%q %v", got, err)
	}
	st, err := l.Stat(p)
	if err != nil || st.IsDir() {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := l.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	p2 := filepath.Join(sub, "b.txt")
	if err := l.Rename(p, p2); err != nil {
		t.Fatal(err)
	}
	if err := l.Remove(p2); err != nil {
		t.Fatal(err)
	}
	if l.Join("a", "b") == "" {
		t.Fatal("join")
	}
	if _, err := l.LookPath("go"); err != nil {
		t.Fatal(err)
	}
	if !IsNotExist(os.ErrNotExist) {
		t.Fatal("IsNotExist")
	}
	data, err := ReadMaybe(l, filepath.Join(dir, "missing"))
	if err != nil || data != nil {
		t.Fatalf("%v %v", data, err)
	}
	if Exists(l, filepath.Join(dir, "missing")) {
		t.Fatal("exists")
	}
	if !ExistsAny(l, dir) {
		t.Fatal("ExistsAny dir")
	}
	target := filepath.Join(dir, "nested", "c.txt")
	if err := AtomicWrite(l, target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !Exists(l, target) {
		t.Fatal("atomic")
	}
	bak, err := BackupLocal(filepath.Join(dir, "bak"), "codex", []byte("src"))
	if err != nil || bak == "" {
		t.Fatal(err)
	}
	empty, err := BackupLocal(filepath.Join(dir, "bak"), "codex", nil)
	if err != nil || empty != "" {
		t.Fatal(err)
	}
	if dirOf(l, target) != filepath.Dir(target) {
		t.Fatal(dirOf(l, target))
	}
	s := SSH{Target: "u@h"}
	if dirOf(s, "/a/b") != "/a" {
		t.Fatal(dirOf(s, "/a/b"))
	}
	if IsNotExist(errors.New("other")) {
		t.Fatal("other")
	}
}
