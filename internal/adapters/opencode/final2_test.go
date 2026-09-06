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

func TestParseApplyNodeNth(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "opencode.json"), []byte("{"), 0o600)
	snap, _ := (Adapter{}).ReadFS(fsx.Local{}, home)
	if snap.ParseError == "" {
		t.Log(snap) // may set parse error
	}
	real := edit.SetJSONCHook
	defer func() { edit.SetJSONCHook = real }()
	for _, nth := range []int{1, 2} {
		n := 0
		edit.SetJSONCHook = func(src []byte, path []string, value string) ([]byte, error) {
			n++
			if n == nth {
				return nil, errors.New("boom")
			}
			return real(src, path, value)
		}
		_, err := (Adapter{}).WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Model: "a/b", Provider: "c", SecretRef: "K"})
		if err == nil {
			t.Fatal(nth)
		}
	}
}
