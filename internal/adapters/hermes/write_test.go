package hermes

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
	_, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "openrouter", SecretRef: "OPENROUTER_API_KEY"})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "OPENROUTER_API_KEY", "sk-test-hermes"); err != nil {
		t.Fatal(err)
	}
	ref, val, err := a.PeekSecret(fsx.Local{}, home)
	if err != nil || val != "sk-test-hermes" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "", "sk-test-2"); err != nil {
		t.Fatal(err)
	}
	empty := t.TempDir()
	_, _, err = a.PeekSecret(fsx.Local{}, empty)
	if err != nil {
		t.Fatal(err)
	}
	// string model empty + dotenv parse
	home2 := t.TempDir()
	os.MkdirAll(filepath.Join(home2, ".hermes"), 0o755)
	os.WriteFile(filepath.Join(home2, ".hermes", "config.yaml"), []byte("model: \"\"\n"), 0o600)
	os.WriteFile(filepath.Join(home2, ".hermes", ".env"), []byte("#c\n\nFOO=bar\nbadline\nOPENAI_API_KEY=sk-test-a\n"), 0o600)
	snap, err := a.ReadFS(fsx.Local{}, home2)
	if err != nil || !snap.SecretPresent {
		t.Fatalf("%#v %v", snap, err)
	}
	m, _ := parseDotEnvBytes([]byte("A=1\n#x\n"))
	if m["A"] != "1" {
		t.Fatal(m)
	}
	// providers single
	home3 := t.TempDir()
	os.MkdirAll(filepath.Join(home3, ".hermes"), 0o755)
	os.WriteFile(filepath.Join(home3, ".hermes", "config.yaml"), []byte(`
model:
  default: m
providers:
  only:
    base_url: https://h.test
    api_key: sk-test-h
`), 0o600)
	snap, err = a.ReadFS(fsx.Local{}, home3)
	if err != nil || snap.Provider != "only" || snap.BaseURLHost == "" {
		t.Fatalf("%#v %v", snap, err)
	}
}
