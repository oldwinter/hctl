package codex

import (
	"errors"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestForcedSetErrors(t *testing.T) {
	old := edit.SetTOMLHook
	defer func() { edit.SetTOMLHook = old }()
	edit.SetTOMLHook = func(src []byte, path []string, value string) ([]byte, error) {
		return nil, errors.New("set boom")
	}
	home := t.TempDir()
	a := Adapter{}
	if _, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "p", SecretRef: "K"}); err == nil {
		t.Fatal("expected")
	}

	if err := a.WriteSecret(fsx.Local{}, home, "", "sk-test-x"); err == nil {
		t.Fatal("ws")
	}
}
