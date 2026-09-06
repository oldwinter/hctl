package hermes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type hermesBoom struct{ fsx.Local }

func (hermesBoom) ReadFile(name string) ([]byte, error) { return nil, errors.New("boom") }

func TestHermesMicro(t *testing.T) {
	a := Adapter{}
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes"), 0o755)
	os.WriteFile(filepath.Join(home, ".hermes", "config.yaml"), []byte("{"), 0o600)
	snap, _ := a.ReadFS(fsx.Local{}, home)
	if snap.ParseError == "" {
		t.Fatal("parse")
	}
	real := edit.SetYAMLHook
	defer func() { edit.SetYAMLHook = real }()
	for _, want := range []int{1, 2, 3} {
		want := want
		n := 0
		edit.SetYAMLHook = func(src []byte, path []string, value string) ([]byte, error) {
			n++
			if n == want {
				return nil, errors.New("boom")
			}
			return real(src, path, value)
		}
		if _, err := a.WriteFields(fsx.Local{}, t.TempDir(), model.Desired{Model: "m", Provider: "p", SecretRef: "K"}); err == nil {
			t.Fatal(want)
		}
	}
	edit.SetYAMLHook = real
	if _, err := a.WriteFields(fsx.Local{}, t.TempDir(), model.Desired{SecretRef: "K"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := a.PeekSecret(hermesBoom{}, home); err == nil {
		t.Fatal("peek")
	}
}

func TestPeekSecretEnvOnly(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".hermes"), 0o755)
	os.WriteFile(filepath.Join(home, ".hermes", "config.yaml"), []byte("model: m\n"), 0o600)
	os.WriteFile(filepath.Join(home, ".hermes", ".env"), []byte("FOO=sk-test-f\n"), 0o600)
	ref, val, err := (Adapter{}).PeekSecret(fsx.Local{}, home)
	if err != nil || ref == "" || val == "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	os.WriteFile(filepath.Join(home, ".hermes", ".env"), []byte("# empty\n"), 0o600)
	ref, val, err = (Adapter{}).PeekSecret(fsx.Local{}, home)
	if err != nil || ref != "" || val != "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
}
