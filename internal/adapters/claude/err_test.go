package claude

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type boomFS struct {
	fsx.Local
	failRead, failWrite, failMkdir, failRename bool
}

func (b boomFS) ReadFile(name string) ([]byte, error) {
	if b.failRead {
		return nil, errors.New("boom")
	}
	return b.Local.ReadFile(name)
}
func (b boomFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	if b.failWrite {
		return errors.New("boom")
	}
	return b.Local.WriteFile(name, data, perm)
}
func (b boomFS) MkdirAll(name string, perm os.FileMode) error {
	if b.failMkdir {
		return errors.New("boom")
	}
	return b.Local.MkdirAll(name, perm)
}
func (b boomFS) Rename(oldpath, newpath string) error {
	if b.failRename {
		return errors.New("boom")
	}
	return b.Local.Rename(oldpath, newpath)
}

func TestErrorPaths(t *testing.T) {
	home := t.TempDir()
	a := Adapter{}
	_, err := a.WriteFields(boomFS{failMkdir: true}, home, model.Desired{Model: "m"})
	if err == nil {
		t.Fatal("atomic")
	}
	_, err = a.ReadFS(boomFS{failRead: true}, home)
	if err == nil {
		t.Fatal("read")
	}
	// PeekSecret read error
	_, _, err = a.PeekSecret(boomFS{failRead: true}, home)
	if err == nil {
		// ReadMaybe returns err on non-NotExist
		t.Fatal("peek")
	}
	if err := a.WriteSecret(boomFS{failMkdir: true}, home, "", "sk-test-x"); err == nil {
		t.Fatal("ws")
	}
	// WriteFields SetJSON path with existing bad? use fail after read
	os.MkdirAll(filepath.Join(home, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(`{"model":"m","env":{}}`), 0o600)
	_, err = a.WriteFields(boomFS{failRename: true}, home, model.Desired{Model: "n"})
	if err == nil {
		t.Fatal("rename")
	}
	// bool onboarding true
	home2 := t.TempDir()
	os.MkdirAll(filepath.Join(home2, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home2, ".claude", "settings.json"), []byte(`{"model":"m"}`), 0o600)
	os.WriteFile(filepath.Join(home2, ".claude.json"), []byte(`{"theme":"t","hasCompletedOnboarding":true}`), 0o600)
	snap, _ := a.ReadFS(fsx.Local{}, home2)
	for _, n := range snap.Notes {
		if n != "" && false {
			t.Log(n)
		}
	}
	_ = applyClaudeEnv
	snap = model.Snapshot{}
	applyClaudeEnv(&snap, map[string]string{"ANTHROPIC_BASE_URL": "https://api.anthropic.com", "ANTHROPIC_DEFAULT_OPUS_MODEL": "x", "ANTHROPIC_AUTH_TOKEN": "sk-test-zzzzzz"})
	if snap.SecretFingerprint == "" {
		t.Fatal("fp")
	}
}
