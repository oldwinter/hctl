package hermes

import (
	"errors"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestForcedSetErrors(t *testing.T) {
	oldY, oldD := edit.SetYAMLHook, edit.SetDotEnvHook
	defer func() {
		edit.SetYAMLHook, edit.SetDotEnvHook = oldY, oldD
	}()
	edit.SetYAMLHook = func(src []byte, path []string, value string) ([]byte, error) {
		return nil, errors.New("set boom")
	}
	home := t.TempDir()
	a := Adapter{}
	if _, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "p", SecretRef: "K"}); err == nil {
		t.Fatal("expected")
	}
	edit.SetDotEnvHook = func(src []byte, key, value string) ([]byte, error) {
		return nil, errors.New("set boom")
	}
	if err := a.WriteSecret(fsx.Local{}, home, "K", "sk-test-x"); err == nil {
		t.Fatal("ws")
	}
}
