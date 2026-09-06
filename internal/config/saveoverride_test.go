package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSetSaveOverride(t *testing.T) {
	SetSaveOverride(func(string, *File) error { return errors.New("x") })
	defer SetSaveOverride(nil)
	if err := Save(filepath.Join(t.TempDir(), "c.yaml"), Default()); err == nil {
		t.Fatal("expected")
	}
	SetSaveOverride(nil)
	if err := Save(filepath.Join(t.TempDir(), "c.yaml"), Default()); err != nil {
		t.Fatal(err)
	}
}
