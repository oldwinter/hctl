package codex

import (
	"errors"
	"os"
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
	_, err := a.WriteFields(boomFS{failMkdir: true}, home, model.Desired{Model: "m", Provider: "custom"})
	if err == nil {
		t.Fatal("atomic")
	}
	_, err = a.ReadFS(boomFS{failRead: true}, home)
	if err == nil {
		t.Fatal("read")
	}

	if err := a.WriteSecret(boomFS{failMkdir: true}, home, "", "sk-test-x"); err == nil {
		t.Fatal("ws")
	}
	_, _, _ = a.PeekSecret(boomFS{failRead: true}, home)

	_ = os.TempDir
	_ = fsx.Local{}
}
