package codex

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestParseErrorAndNthSetTOML(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".codex"), 0o755)
	os.WriteFile(filepath.Join(home, ".codex", "config.toml"), []byte("[[["), 0o600)
	snap, err := (Adapter{}).ReadFS(fsx.Local{}, home)
	if err != nil || snap.ParseError == "" {
		t.Fatalf("%#v %v", snap, err)
	}
	real := edit.SetTOMLHook
	defer func() { edit.SetTOMLHook = real }()
	for _, want := range []int{1, 2, 3} {
		want := want
		n := 0
		edit.SetTOMLHook = func(src []byte, path []string, value string) ([]byte, error) {
			n++
			if n == want {
				return nil, errors.New("boom")
			}
			return real(src, path, value)
		}
		_, err := (Adapter{}).WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Model: "m", Provider: "p", SecretRef: "K"})
		if err == nil {
			t.Fatalf("nth=%d", want)
		}
	}
	n := 0
	edit.SetTOMLHook = func(src []byte, path []string, value string) ([]byte, error) {
		n++
		if n == 2 {
			return nil, errors.New("boom")
		}
		return real(src, path, value)
	}
	if err := (Adapter{}).WriteSecret(fsx.Local{}, t.TempDir(), "K", "sk-test-x"); err == nil {
		t.Fatal("ws")
	}
}
