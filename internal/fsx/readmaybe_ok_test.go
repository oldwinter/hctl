package fsx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadMaybeSuccess(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	os.WriteFile(p, []byte("hi"), 0o600)
	data, err := ReadMaybe(Local{}, p)
	if err != nil || string(data) != "hi" {
		t.Fatalf("%q %v", data, err)
	}
}
