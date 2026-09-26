package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBackupLocalReservesUniqueSourceIdentifiedFiles(t *testing.T) {
	dir := t.TempDir()
	first, err := BackupLocal(dir, "pi", "/home/fixture/.pi/agent/settings.json", []byte("settings-original"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := BackupLocal(dir, "pi", "/home/fixture/.pi/agent/models.json", []byte("models-original"))
	if err != nil {
		t.Fatal(err)
	}
	third, err := BackupLocal(dir, "pi", "/home/fixture/.pi/agent/settings.json", []byte("settings-next"))
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first == third || second == third {
		t.Fatalf("backup collision: %q %q %q", first, second, third)
	}
	for path, want := range map[string]string{first: "settings-original", second: "models-original", third: "settings-next"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Fatalf("%s = %q want %q", filepath.Base(path), data, want)
		}
		if !strings.HasSuffix(path, ".bak") {
			t.Fatalf("backup suffix missing: %s", path)
		}
	}
}

func TestWithoutCommandProbesPreservesLocalAtomicWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	if err := AtomicWrite(WithoutCommandProbes(Local{}), path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "{}\n" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestAtomicWriteMustNotFollowStaleTempSymlink(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	unrelated := filepath.Join(dir, "unrelated.txt")
	if err := os.WriteFile(unrelated, []byte("keep-me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("old-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(unrelated, dest+".harnessctl-tmp"); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(Local{}, dest, []byte("new-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(unrelated); err != nil || string(data) != "keep-me" {
		t.Fatalf("unrelated=%q err=%v", data, err)
	}
	st, err := os.Lstat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode()&os.ModeSymlink != 0 {
		t.Fatal("destination became a symlink")
	}
	if data, err := os.ReadFile(dest); err != nil || string(data) != "new-config" {
		t.Fatalf("dest=%q err=%v", data, err)
	}
}

func TestAtomicWriteMustNotFollowDanglingTempSymlink(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	victim := filepath.Join(dir, "victim.txt")
	// Any pre-created harnessctl-tmp* sibling must never be opened for writing.
	matches, err := filepath.Glob(dest + ".harnessctl-tmp*")
	if err != nil || len(matches) != 0 {
		t.Fatalf("unexpected pre-existing temp files: %v", matches)
	}
	if err := os.Symlink(victim, dest+".harnessctl-tmp"); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(Local{}, dest, []byte("new-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(victim); !os.IsNotExist(err) {
		t.Fatalf("dangling symlink target was created: %v", err)
	}
	if data, err := os.ReadFile(dest); err != nil || string(data) != "new-config" {
		t.Fatalf("dest=%q err=%v", data, err)
	}
}

func TestAtomicWriteLeavesPreexistingTempSiblingUntouched(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	stale := dest + ".harnessctl-tmp"
	if err := os.WriteFile(stale, []byte("other-writer"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(Local{}, dest, []byte("new-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(stale); err != nil || string(data) != "other-writer" {
		t.Fatalf("stale temp sibling overwritten: %q err=%v", data, err)
	}
	st, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Mode().Perm(); got != 0o600 {
		t.Fatalf("dest perm=%o want 600", got)
	}
	if data, err := os.ReadFile(dest); err != nil || string(data) != "new-config" {
		t.Fatalf("dest=%q err=%v", data, err)
	}
}

func TestAtomicWriteInterleavedWriters(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	payloads := map[string]bool{}
	var wg sync.WaitGroup
	for i := range 8 {
		payload := fmt.Sprintf("writer-%d-config-with-padding", i)
		payloads[payload] = true
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := AtomicWrite(Local{}, dest, []byte(payload), 0o600); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !payloads[string(data)] {
		t.Fatalf("interleaved writers produced %q", data)
	}
	assertNoTempResidue(t, dir)
}

type failRenameFS struct{ Local }

func (failRenameFS) Rename(oldpath, newpath string) error {
	return errors.New("rename denied")
}

func TestAtomicWriteRemovesTempOnRenameFailure(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	if err := AtomicWrite(failRenameFS{}, dest, []byte("new-config"), 0o600); err == nil {
		t.Fatal("expected rename failure")
	}
	assertNoTempResidue(t, dir)
}

type failWriteFS struct{ Local }

func (f failWriteFS) WriteNewFile(name string, data []byte, perm os.FileMode) error {
	if err := f.Local.WriteNewFile(name, []byte("partial"), perm); err != nil {
		return err
	}
	return errors.New("write failed midway")
}

func TestAtomicWriteRemovesTempOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "config.json")
	if err := AtomicWrite(failWriteFS{}, dest, []byte("new-config"), 0o600); err == nil {
		t.Fatal("expected write failure")
	}
	assertNoTempResidue(t, dir)
}

func TestLocalWriteNewFileRejectsExisting(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (Local{}).WriteNewFile(existing, []byte("evil"), 0o600); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("existing file: err=%v want ErrExist", err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(existing, link); err != nil {
		t.Fatal(err)
	}
	if err := (Local{}).WriteNewFile(link, []byte("evil"), 0o600); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("symlink: err=%v want ErrExist", err)
	}
	dangling := filepath.Join(dir, "dangling")
	if err := os.Symlink(filepath.Join(dir, "nowhere"), dangling); err != nil {
		t.Fatal(err)
	}
	if err := (Local{}).WriteNewFile(dangling, []byte("evil"), 0o600); !errors.Is(err, fs.ErrExist) {
		t.Fatalf("dangling symlink: err=%v want ErrExist", err)
	}
	if data, err := os.ReadFile(existing); err != nil || string(data) != "keep" {
		t.Fatalf("target=%q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "nowhere")); !os.IsNotExist(err) {
		t.Fatalf("dangling target created: %v", err)
	}
}

func assertNoTempResidue(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.harnessctl-tmp*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temp files left behind: %v", matches)
	}
}
