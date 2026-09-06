package desired

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFormats(t *testing.T) {
	p := filepath.Join(t.TempDir(), "d.yaml")
	os.WriteFile(p, []byte("harnesses:\n  codex:\n    model: m\n"), 0o600)
	f, err := Load(p)
	if err != nil || f.Harnesses["codex"].Model != "m" {
		t.Fatal(err, f)
	}
	f, err = Parse([]byte("harnesses = { codex = { model = \"m\" } }\n"), ".toml")
	if err != nil {
		t.Fatal(err)
	}
	_ = f
	f, err = Parse([]byte("harnesses:\n  x:\n    model: y\n"), ".yml")
	if err != nil {
		t.Fatal(err)
	}
	f, err = Parse([]byte("not valid {{{"), "")
	if err == nil {
		t.Fatal("expected")
	}
	f, err = Parse([]byte("harnesses = {}\n"), "")
	if err != nil {
		t.Fatal(err)
	}
	if f.APIVersion == "" || f.Kind == "" {
		t.Fatal(f)
	}
}
