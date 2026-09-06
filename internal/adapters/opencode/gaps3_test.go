package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestApplyNodeEmptyKeyReturn(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"p/m","provider":{"p":{"options":{"baseURL":"https://example.com"}}}}`), 0o600)
	snap, err := (Adapter{}).ReadFS(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.BaseURLHost == "" {
		t.Fatalf("%+v", snap)
	}
}

func TestWriteFieldsSecretRefFromModelSlash(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"old/m"}`), 0o600)
	_, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "API_KEY", Model: "prov/model"})
	if err != nil {
		t.Fatal(err)
	}
}
