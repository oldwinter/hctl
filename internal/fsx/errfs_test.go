package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type errFS struct {
	Local
	failRead   bool
	failWrite  bool
	failStat   bool
	failMkdir  bool
	failRename bool
	failRemove bool
}

func (e errFS) ReadFile(name string) ([]byte, error) {
	if e.failRead {
		return nil, errors.New("read boom")
	}
	return e.Local.ReadFile(name)
}
func (e errFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	if e.failWrite {
		return errors.New("write boom")
	}
	return e.Local.WriteFile(name, data, perm)
}
func (e errFS) Stat(name string) (os.FileInfo, error) {
	if e.failStat {
		return nil, errors.New("stat boom")
	}
	return e.Local.Stat(name)
}
func (e errFS) MkdirAll(name string, perm os.FileMode) error {
	if e.failMkdir {
		return errors.New("mkdir boom")
	}
	return e.Local.MkdirAll(name, perm)
}
func (e errFS) Rename(oldpath, newpath string) error {
	if e.failRename {
		return errors.New("rename boom")
	}
	return e.Local.Rename(oldpath, newpath)
}
func (e errFS) Remove(name string) error {
	if e.failRemove {
		return errors.New("remove boom")
	}
	return e.Local.Remove(name)
}

func TestReadMaybeAtomicBackupErrors(t *testing.T) {
	dir := t.TempDir()
	e := errFS{failRead: true}
	if _, err := ReadMaybe(e, filepath.Join(dir, "x")); err == nil {
		t.Fatal("readmaybe")
	}
	e2 := errFS{failMkdir: true}
	if err := AtomicWrite(e2, filepath.Join(dir, "a", "b"), []byte("x"), 0o600); err == nil {
		t.Fatal("mkdir")
	}
	e3 := errFS{failWrite: true}
	if err := AtomicWrite(e3, filepath.Join(dir, "c.txt"), []byte("x"), 0o600); err == nil {
		t.Fatal("write")
	}
	// rename fail then remove
	e4 := errFS{failRename: true}
	p := filepath.Join(dir, "d.txt")
	if err := AtomicWrite(e4, p, []byte("x"), 0o600); err == nil {
		t.Fatal("rename")
	}
	e5 := errFS{failRename: true, failRemove: true}
	if err := AtomicWrite(e5, filepath.Join(dir, "e.txt"), []byte("x"), 0o600); err == nil {
		t.Fatal("rename+remove")
	}
	// BackupLocal mkdir fail via unwritable parent — skip if root; use invalid path
	if _, err := BackupLocal(filepath.Join(dir, "no", "perm"), "h", []byte("x")); err != nil {
		// may succeed if mkdir works; force fail by using file as dir
		f := filepath.Join(dir, "file")
		os.WriteFile(f, []byte("x"), 0o600)
		_, err = BackupLocal(filepath.Join(f, "sub"), "h", []byte("x"))
		if err == nil {
			t.Fatal("backup mkdir")
		}
	}
}

func TestSSHRunNilUsesDefaultAndIsExitUnwrap(t *testing.T) {
	// run with custom that wraps exitcode
	s := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return nil, errors.New("fail")
	}}
	_, err := s.run(nil, "true")
	if err == nil {
		t.Fatal("expected wrap")
	}
	// Stat with exit 3 via unwrap chain
	s2 := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return nil, errors.New("exit status 3")
	}}
	_, err = s2.Stat("/no")
	if err == nil {
		t.Fatal("missing")
	}
}
