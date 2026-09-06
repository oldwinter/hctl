package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSaveMarshalError(t *testing.T) {
	old := yamlMarshal
	defer func() { yamlMarshal = old }()
	yamlMarshal = func(v any) ([]byte, error) { return nil, errors.New("marshal") }
	if err := Save(filepath.Join(t.TempDir(), "c.yaml"), Default()); err == nil {
		t.Fatal("expected")
	}
}
