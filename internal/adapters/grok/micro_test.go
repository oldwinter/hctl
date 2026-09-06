package grok

import (
	"errors"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestMicroBranches(t *testing.T) {
	a := Adapter{}
	home := t.TempDir()
	// SecretRef only -> name default
	_, err := a.WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "K"})
	if err != nil {
		t.Fatal(err)
	}
	ref, _, err := a.PeekSecret(fsx.Local{}, home)
	_ = ref
	// Peek with models but nil Model map
	real := edit.SetTOMLHook
	defer func() { edit.SetTOMLHook = real }()
	edit.SetTOMLHook = real
	home2 := t.TempDir()
	_, _ = a.WriteFields(fsx.Local{}, home2, model.Desired{Model: "m"})
	// WriteSecret value then ref with empty default model name path
	home3 := t.TempDir()
	if err := a.WriteSecret(fsx.Local{}, home3, "K", "sk-test-g"); err != nil {
		t.Fatal(err)
	}
	if err := a.WriteSecret(fsx.Local{}, home3, "K2", ""); err != nil {
		t.Fatal(err)
	}
	n := 0
	edit.SetTOMLHook = func(src []byte, path []string, value string) ([]byte, error) {
		n++
		if n == 1 && path[len(path)-1] == "env_key" {
			return nil, errors.New("boom")
		}
		return real(src, path, value)
	}
	_ = a.WriteSecret(fsx.Local{}, t.TempDir(), "K", "")
	_ = a.WriteSecret(fsx.Local{}, t.TempDir(), "K", "sk-test-g")
}
