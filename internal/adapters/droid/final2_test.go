package droid

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestParseEnvRefAndNth(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".factory"), 0o755)
	os.WriteFile(filepath.Join(home, ".factory", "settings.json"), []byte("{"), 0o600)
	snap, _ := (Adapter{}).ReadFS(fsx.Local{}, home)
	if snap.ParseError == "" {
		t.Fatal("parse")
	}
	os.WriteFile(filepath.Join(home, ".factory", "settings.json"), []byte(`{"apiKey":"$DROID_KEY","model":"m"}`), 0o600)
	snap, _ = (Adapter{}).ReadFS(fsx.Local{}, home)
	if snap.SecretRef == "" {
		t.Fatalf("%#v", snap)
	}
	real := edit.SetJSONHook
	defer func() { edit.SetJSONHook = real }()
	n := 0
	edit.SetJSONHook = func(src []byte, path []string, value string) ([]byte, error) {
		n++
		if n == 2 {
			return nil, errors.New("boom")
		}
		return real(src, path, value)
	}
	_, err := (Adapter{}).WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Model: "m", Provider: "p"})
	if err == nil {
		t.Fatal("nth2")
	}
}
