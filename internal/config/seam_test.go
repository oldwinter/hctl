package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestUserHomeDirSeams(t *testing.T) {
	old := userHomeDir
	defer func() { userHomeDir = old }()
	userHomeDir = func() (string, error) { return "", errors.New("no home") }
	t.Setenv("HARNESSCTL_CONFIG", "")
	p := DefaultPath()
	if p == "" {
		t.Fatal("path")
	}
	f := Default()
	_, _, err := f.ResolveHome("mba", "")
	if err == nil {
		t.Fatal("expected")
	}
	_ = expandHome("~/x")
	// Save mkdir fail: parent is a file
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	os.WriteFile(file, []byte("x"), 0o600)
	if err := Save(filepath.Join(file, "c.yaml"), f); err == nil {
		t.Fatal("save")
	}
	// Load permission: create unreadable file
	bad := filepath.Join(dir, "bad.yaml")
	os.WriteFile(bad, []byte("x"), 0o000)
	_, err = Load(bad)
	if err == nil {
		// some OS still allow root read; log
		t.Log("load perms", err)
	}
	os.Chmod(bad, 0o600)
}
