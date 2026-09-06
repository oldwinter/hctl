package config

import (
	"errors"
	"testing"
)

func TestResolveHomeUserHomeFail(t *testing.T) {
	old := userHomeDir
	defer func() { userHomeDir = old }()
	userHomeDir = func() (string, error) { return "", errors.New("no") }
	f := Default()
	_, _, err := f.ResolveHome("mba", "")
	if err == nil {
		t.Fatal("expected")
	}
}

func TestResolveHomeMissingContext(t *testing.T) {
	f := Default()
	if _, _, err := f.ResolveHome("nope", ""); err == nil {
		t.Fatal("expected")
	}
}
