package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestMetaWritePeekSecret(t *testing.T) {
	a := Adapter{}
	_ = a.Aliases()
	_ = a.BinaryNames()
	_ = a.ConfigRelPaths()
	home := t.TempDir()
	_, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "openai/gpt", Provider: "anthropic", SecretRef: "KEY"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "KEY", "sk-test-oc"); err != nil {
		t.Fatal(err)
	}
	_, _, err = a.PeekSecret(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	home2 := t.TempDir()
	_, err = a.WriteFields(fsx.Local{}, home2, model.Desired{Provider: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	home3 := t.TempDir()
	dir := filepath.Join(home3, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"openai/gpt-4","provider":{"openai":{"options":{"apiKey":"sk-test-a"}}}}`), 0o600)
	ref, val, err := a.PeekSecret(fsx.Local{}, home3)
	if err != nil || val != "sk-test-a" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	_, err = a.WriteFields(fsx.Local{}, home3, model.Desired{SecretRef: "OPENAI_API_KEY"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.WriteSecret(fsx.Local{}, home3, "OPENAI_API_KEY", ""); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	_, _, err = a.PeekSecret(fsx.Local{}, empty)
	if err != nil {
		t.Fatal(err)
	}
	snap, _ := a.Read(t.TempDir())
	if snap.ConfigFound {
		t.Fatal("missing")
	}
	// providers + settings + api_key alt + jsonc comments
	home4 := t.TempDir()
	dir = filepath.Join(home4, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte(`{"model":"acme/m","providers":{"acme":{"settings":{"api_key":"$ACME_KEY"}}}}`), 0o600)
	snap, err = a.ReadFS(fsx.Local{}, home4)
	if err != nil {
		t.Fatal(err)
	}
	ref, val, err = a.PeekSecret(fsx.Local{}, home4)
	if err != nil || ref == "" {
		t.Fatalf("%q %q %v snap=%#v", ref, val, err, snap)
	}
}
