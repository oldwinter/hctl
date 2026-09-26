package fsx

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
	// WriteNewFile creates name exclusively and fails with fs.ErrExist when
	// the name is taken. It never follows a planted symlink.
	WriteNewFile(name string, data []byte, perm os.FileMode) error
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
func (Local) WriteNewFile(name string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
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

type noCommandProbes struct {
	FS
}

func (n noCommandProbes) LookPath(name string) (string, error) {
	looker, ok := n.FS.(interface {
		LookPath(string) (string, error)
	})
	if !ok {
		return "", os.ErrNotExist
	}
	return looker.LookPath(name)
}

func (noCommandProbes) CommandProbesAllowed() bool { return false }

// WithoutCommandProbes keeps filesystem access and PATH lookup but disables
// version/login subprocesses used only to enrich inventory.
func WithoutCommandProbes(fsys FS) FS {
	return noCommandProbes{FS: fsys}
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

// AtomicWrite writes via a uniquely reserved sibling temp file then rename.
func AtomicWrite(fsys FS, name string, data []byte, perm os.FileMode) error {
	if err := fsys.MkdirAll(dirOf(fsys, name), 0o700); err != nil {
		return err
	}
	var tmp string
	var err error
	for range 5 {
		tmp = name + ".harnessctl-tmp." + randSuffix()
		if err = fsys.WriteNewFile(tmp, data, perm); err == nil {
			break
		}
		if !errors.Is(err, fs.ErrExist) {
			_ = fsys.Remove(tmp)
			return err
		}
	}
	if err != nil {
		return err
	}
	if err := fsys.Rename(tmp, name); err != nil {
		_ = fsys.Remove(tmp)
		return err
	}
	return nil
}

func randSuffix() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// BackupLocal copies src bytes into a uniquely reserved local file (0600).
func BackupLocal(destDir, harness, sourcePath string, src []byte) (string, error) {
	if len(src) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(destDir, 0o700); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(sourcePath))
	source := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(filepath.Base(sourcePath))
	prefix := fmt.Sprintf("%s-%s-%s-%s-", harness, source, hex.EncodeToString(sum[:4]), time.Now().UTC().Format("20060102T150405.000000000Z"))
	f, err := os.CreateTemp(destDir, prefix+"*.bak")
	if err != nil {
		return "", err
	}
	backupPath := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(backupPath)
		}
	}()
	if err := f.Chmod(0o600); err != nil {
		return "", err
	}
	if _, err := f.Write(src); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	ok = true
	return backupPath, nil
}

func dirOf(fsys FS, name string) string {
	switch typed := fsys.(type) {
	case Local:
		return filepath.Dir(name)
	case noCommandProbes:
		return dirOf(typed.FS, name)
	}
	return path.Dir(strings.ReplaceAll(name, "\\", "/"))
}
