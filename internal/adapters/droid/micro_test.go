package droid

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
)

type boom struct{ fsx.Local }

func (boom) ReadFile(string) ([]byte, error) { return nil, errors.New("x") }

func TestReadFSOtherError(t *testing.T) {
	if _, err := (Adapter{}).ReadFS(boom{}, t.TempDir()); err == nil {
		t.Fatal("expected")
	}
}

func TestReadLiteralAPIKey(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".factory"), 0o755)
	os.WriteFile(filepath.Join(home, ".factory", "settings.json"), []byte(`{"model":"m","apiKey":"sk-test-lit"}`), 0o600)
	snap, err := (Adapter{}).ReadFS(fsx.Local{}, home)
	if err != nil || !snap.SecretPresent {
		t.Fatalf("%+v %v", snap, err)
	}
}

func TestReadMissingAuthNote(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".factory"), 0o755)
	os.WriteFile(filepath.Join(home, ".factory", "settings.json"), []byte(`{"model":"m"}`), 0o600)
	snap, err := (Adapter{}).ReadFS(fsx.Local{}, home)
	if err != nil || len(snap.Notes) == 0 {
		t.Fatalf("%+v %v", snap, err)
	}
}
