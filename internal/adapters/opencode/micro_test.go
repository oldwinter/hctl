package opencode

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestApplyNodeVariants(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"p/m","provider":{"p":{"options":{"baseUrl":"https://a.test","reasoningEffort":"high","apiKey":"$K"}}}}`), 0o600)
	snap, err := (Adapter{}).ReadFS(fsx.Local{}, home)
	if err != nil || snap.SecretRef == "" || snap.Effort == "" {
		t.Fatalf("%#v %v", snap, err)
	}
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"m","providers":{"only":{"settings":{"base_url":"https://b.test","api_key":"sk-test-oc"}}}}`), 0o600)
	snap, err = (Adapter{}).ReadFS(fsx.Local{}, home)
	if err != nil || !snap.SecretPresent {
		t.Fatalf("%#v %v", snap, err)
	}
	_, err = (Adapter{}).WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Provider: "openai"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = (Adapter{}).WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Provider: "openai", Model: "gpt"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "K"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMoreOpencode(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"p/m","provider":{"p":{}}}`), 0o600)
	snap, _ := (Adapter{}).ReadFS(fsx.Local{}, home)
	_ = snap
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"p/m","provider":{"p":{"options":{"apiKey":"sk-test-z"}}}}`), 0o600)
	ref, val, err := (Adapter{}).PeekSecret(fsx.Local{}, home)
	if err != nil || val == "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"alone"}`), 0o600)
	ref, val, err = (Adapter{}).PeekSecret(fsx.Local{}, home)
	if err != nil || ref != "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	// WriteFields provider with existing model containing slash
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte(`{"model":"old/m"}`), 0o600)
	_, err = (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Provider: "new"})
	if err != nil {
		t.Fatal(err)
	}
}
