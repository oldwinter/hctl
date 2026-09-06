package fsx

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// FS is the small file API used by readers and writers (local or SSH).
type FS interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	Stat(name string) (os.FileInfo, error)
	MkdirAll(name string, perm os.FileMode) error
	Rename(oldpath, newpath string) error
	Remove(name string) error
	// Join builds a path on this filesystem (OS-aware locally, slash-separated remotely).
	Join(elem ...string) string
}

// Local is the host filesystem.
type Local struct{}

func (Local) ReadFile(name string) ([]byte, error) { return os.ReadFile(name) }
func (Local) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}
func (Local) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (Local) MkdirAll(name string, perm os.FileMode) error {
	return os.MkdirAll(name, perm)
}
func (Local) Rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }
func (Local) Remove(name string) error             { return os.Remove(name) }
func (Local) Join(elem ...string) string           { return filepath.Join(elem...) }

// LookPath resolves a binary on the local PATH.
func (Local) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, fs.ErrNotExist)
}

// ReadMaybe returns (nil, nil) when the file does not exist.
func ReadMaybe(fsys FS, name string) ([]byte, error) {
	data, err := fsys.ReadFile(name)
	if err != nil {
		if IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func Exists(fsys FS, name string) bool {
	st, err := fsys.Stat(name)
	return err == nil && !st.IsDir()
}

func ExistsAny(fsys FS, name string) bool {
	_, err := fsys.Stat(name)
	return err == nil
}

// AtomicWriteHook is overridden in tests.
var AtomicWriteHook = atomicWriteImpl

// AtomicWrite writes via a sibling temp file then rename.
func AtomicWrite(fsys FS, name string, data []byte, perm os.FileMode) error {
	return AtomicWriteHook(fsys, name, data, perm)
}

func atomicWriteImpl(fsys FS, name string, data []byte, perm os.FileMode) error {
	if err := fsys.MkdirAll(dirOf(fsys, name), 0o700); err != nil {
		return err
	}
	tmp := name + ".harnessctl-tmp"
	if err := fsys.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := fsys.Rename(tmp, name); err != nil {
		_ = fsys.Remove(tmp)
		return err
	}
	return nil
}

// BackupLocal copies src bytes into destDir as harness-timestamp.bak (0600).
func BackupLocal(destDir, harness string, src []byte) (string, error) {
	if len(src) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", err
	}
	name := harness + "-" + time.Now().UTC().Format("20060102T150405Z") + ".bak"
	path := filepath.Join(destDir, name)
	if err := os.WriteFile(path, src, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func dirOf(fsys FS, name string) string {
	if _, ok := fsys.(Local); ok {
		return filepath.Dir(name)
	}
	return path.Dir(strings.ReplaceAll(name, "\\", "/"))
}
