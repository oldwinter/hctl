package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveWriteFail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	dir := t.TempDir()
	os.Chmod(dir, 0o555)
	defer os.Chmod(dir, 0o755)
	if err := Save(filepath.Join(dir, "c.yaml"), Default()); err == nil {
		t.Fatal("expected")
	}
}
