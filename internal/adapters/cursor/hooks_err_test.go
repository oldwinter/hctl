package cursor

import (
	"errors"
	"testing"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestForcedSetErrors(t *testing.T) {
	old := edit.SetJSONHook
	defer func() { edit.SetJSONHook = old }()
	edit.SetJSONHook = func(src []byte, path []string, value string) ([]byte, error) {
		return nil, errors.New("set boom")
	}
	home := t.TempDir()
	a := Adapter{}
	if _, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "m", Provider: "p"}); err == nil {
		t.Fatal("expected")
	}
}
