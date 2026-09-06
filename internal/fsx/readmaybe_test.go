package fsx

import (
	"errors"
	"testing"
)

type alwaysErr struct{ Local }

func (alwaysErr) ReadFile(name string) ([]byte, error) { return nil, errors.New("x") }

func TestReadMaybeOtherError(t *testing.T) {
	if _, err := ReadMaybe(alwaysErr{}, "x"); err == nil {
		t.Fatal("expected")
	}
}
