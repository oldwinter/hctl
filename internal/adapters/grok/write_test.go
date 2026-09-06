package grok

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestMetaWritePeekSecret(t *testing.T) {
	a := Adapter{}
	_ = a.Aliases()
	_ = a.BinaryNames()
	_ = a.ConfigRelPaths()
	home := t.TempDir()
	_, err := a.WriteFields(fsx.Local{}, home, model.Desired{Provider: "x"})
	if err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("%v", err)
	}
	_, err = a.WriteFields(fsx.Local{}, home, model.Desired{Model: "grok-3", SecretRef: "XAI_API_KEY"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "XAI_API_KEY", "sk-test-grok"); err != nil {
		t.Fatal(err)
	}
	ref, val, err := a.PeekSecret(fsx.Local{}, home)
	if err != nil || val != "sk-test-grok" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	empty := t.TempDir()
	_, _, err = a.PeekSecret(fsx.Local{}, empty)
	if err != nil {
		t.Fatal(err)
	}
	bad := t.TempDir()
	os.MkdirAll(filepath.Join(bad, ".grok"), 0o755)
	os.WriteFile(filepath.Join(bad, ".grok", "config.toml"), []byte("[[["), 0o600)
	if _, _, err := a.PeekSecret(fsx.Local{}, bad); err == nil {
		t.Fatal("expected")
	}
	// firstEnvKey slice + stringFromMap + provider inference
	home2 := t.TempDir()
	os.MkdirAll(filepath.Join(home2, ".grok"), 0o755)
	os.WriteFile(filepath.Join(home2, ".grok", "config.toml"), []byte(`
[models]
default = "grok-3"
[model.grok-3]
base_url = "https://api.x.ai/v1"
api_key = "sk-test-x"
env_key = ["XAI_API_KEY", "OTHER"]
reasoning_effort = "high"
`), 0o600)
	snap, err := a.ReadFS(fsx.Local{}, home2)
	if err != nil || snap.Provider != "xai" || snap.Effort != "high" || snap.SecretRef != "XAI_API_KEY" {
		t.Fatalf("%#v %v", snap, err)
	}
	ref, val, err = a.PeekSecret(fsx.Local{}, home2)
	if err != nil || ref != "XAI_API_KEY" || val != "sk-test-x" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	home3 := t.TempDir()
	os.MkdirAll(filepath.Join(home3, ".grok"), 0o755)
	os.WriteFile(filepath.Join(home3, ".grok", "config.toml"), []byte(`
[models]
default = "m"
[model.m]
base_url = "https://custom.example/v1"
env_key = "ENV"
`), 0o600)
	snap, _ = a.ReadFS(fsx.Local{}, home3)
	if snap.Provider != "custom" {
		t.Fatalf("%#v", snap)
	}
	if firstEnvKey(123) != "" || stringFromMap(nil, "x") != "" || stringFromMap(map[string]any{"x": 1}, "x") != "" {
		t.Fatal("helpers")
	}
}
