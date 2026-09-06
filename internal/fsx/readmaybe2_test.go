package fsx

import (
	"errors"
	"testing"
)

type errRead struct{ Local }

func (errRead) ReadFile(string) ([]byte, error) { return nil, errors.New("not-not-exist") }

func TestReadMaybeNonNotExist(t *testing.T) {
	data, err := ReadMaybe(errRead{}, "anything")
	if data != nil || err == nil {
		t.Fatalf("%v %v", data, err)
	}
}
