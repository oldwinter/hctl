package codex

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestMetaWritePeekSecret(t *testing.T) {
	a := Adapter{}
	if a.Aliases()[0] != "openai-codex" {
		t.Fatal(a.Aliases())
	}
	_ = a.BinaryNames()
	_ = a.ConfigRelPaths()
	home := t.TempDir()
	paths, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "o4-mini", Provider: "custom", SecretRef: "OPENAI_API_KEY"})
	if err != nil || len(paths) != 1 {
		t.Fatalf("%v %v", paths, err)
	}
	ref, val, err := a.PeekSecret(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if ref != "OPENAI_API_KEY" {
		t.Fatalf("%q %q", ref, val)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "OPENAI_API_KEY", "sk-test-codex"); err != nil {
		t.Fatal(err)
	}
	ref, val, err = a.PeekSecret(fsx.Local{}, home)
	if err != nil || val != "sk-test-codex" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	empty := t.TempDir()
	ref, val, err = a.PeekSecret(fsx.Local{}, empty)
	if err != nil || ref != "" || val != "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	bad := t.TempDir()
	os.MkdirAll(filepath.Join(bad, ".codex"), 0o755)
	os.WriteFile(filepath.Join(bad, ".codex", "config.toml"), []byte("[[["), 0o600)
	if _, _, err := a.PeekSecret(fsx.Local{}, bad); err == nil {
		t.Fatal("expected err")
	}
	// WriteFields with only SecretRef (provider from snap / custom)
	home2 := t.TempDir()
	_, err = a.WriteFields(fsx.Local{}, home2, model.Desired{SecretRef: "K"})
	if err != nil {
		t.Fatal(err)
	}
	// openai_base_url path + missing providers map in peek
	home3 := t.TempDir()
	os.MkdirAll(filepath.Join(home3, ".codex"), 0o755)
	os.WriteFile(filepath.Join(home3, ".codex", "config.toml"), []byte("model = \"m\"\nopenai_base_url = \"https://api.openai.com/v1\"\n"), 0o600)
	snap, err := a.ReadFS(fsx.Local{}, home3)
	if err != nil || snap.BaseURLHost == "" {
		t.Fatalf("%#v %v", snap, err)
	}
	ref, val, err = a.PeekSecret(fsx.Local{}, home3)
	if err != nil || ref != "" || val != "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
}
