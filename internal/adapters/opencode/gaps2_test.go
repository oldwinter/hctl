package opencode

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestApplyNodeAPIKeySnake(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"p/m","provider":{"p":{"options":{"api_key":"sk-test-snake"}}}}`), 0o600)
	snap, err := (Adapter{}).ReadFS(fsx.Local{}, home)
	if err != nil || !snap.SecretPresent {
		t.Fatalf("%+v %v", snap, err)
	}
}

func TestWriteFieldsSecretSetFail(t *testing.T) {
	prev := edit.SetJSONCHook
	t.Cleanup(func() { edit.SetJSONCHook = prev })
	edit.SetJSONCHook = func(src []byte, path []string, value string) ([]byte, error) {
		return nil, errors.New("set")
	}
	home := t.TempDir()
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "K", Provider: "p"}); err == nil {
		t.Fatal("expected")
	}
}

func TestPeekSecretBadJSON(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{`), 0o600)
	if _, _, err := (Adapter{}).PeekSecret(fsx.Local{}, home); err == nil {
		t.Fatal("expected")
	}
}
