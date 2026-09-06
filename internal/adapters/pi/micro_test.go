package pi

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestMicroReadBranches(t *testing.T) {
	a := Adapter{}
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".pi", "agent"), 0o755)
	os.WriteFile(filepath.Join(home, ".pi", "agent", "settings.json"), []byte("{"), 0o600)
	snap, _ := a.ReadFS(fsx.Local{}, home)
	if snap.ParseError == "" {
		t.Fatal("parse")
	}
	os.WriteFile(filepath.Join(home, ".pi", "agent", "settings.json"), []byte(`{"defaultModel":"dm","base_url":"https://x.test","provider":"p"}`), 0o600)
	os.WriteFile(filepath.Join(home, ".pi", "agent", "auth.json"), []byte(`{"apiKey":"$PI_KEY"}`), 0o600)
	snap, _ = a.ReadFS(fsx.Local{}, home)
	if snap.DefaultModel != "dm" || snap.SecretRef == "" {
		t.Fatalf("%#v", snap)
	}
	os.WriteFile(filepath.Join(home, ".pi", "agent", "settings.json"), []byte(`{"model":"m","baseURL":"https://y.test"}`), 0o600)
	os.WriteFile(filepath.Join(home, ".pi", "agent", "auth.json"), []byte(`{"accessToken":"sk-test-tok"}`), 0o600)
	snap, _ = a.ReadFS(fsx.Local{}, home)
	if !snap.SecretPresent {
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
	if _, err := a.WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Model: "m", Provider: "p"}); err == nil {
		t.Fatal("nth")
	}
}
