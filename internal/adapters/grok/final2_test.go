package grok

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestParseAndNth(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".grok"), 0o755)
	os.WriteFile(filepath.Join(home, ".grok", "config.toml"), []byte("[[["), 0o600)
	snap, _ := (Adapter{}).ReadFS(fsx.Local{}, home)
	if snap.ParseError == "" {
		t.Fatal("parse")
	}
	real := edit.SetTOMLHook
	defer func() { edit.SetTOMLHook = real }()
	for _, nth := range []int{1, 2} {
		n := 0
		edit.SetTOMLHook = func(src []byte, path []string, value string) ([]byte, error) {
			n++
			if n == nth {
				return nil, errors.New("boom")
			}
			return real(src, path, value)
		}
		_, err := (Adapter{}).WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Model: "m", SecretRef: "K"})
		if err == nil {
			t.Fatal(nth)
		}
	}
	if stringFromMap(map[string]any{}, "missing") != "" {
		t.Fatal("missing")
	}
}
