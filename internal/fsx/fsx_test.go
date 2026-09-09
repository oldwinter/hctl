package fsx

import (
	"os"
	"path/filepath"
	"strings"
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
