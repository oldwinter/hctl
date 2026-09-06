package fsx

import (
	"errors"
	"testing"
)

func TestRemoteHomeRunError(t *testing.T) {
	s := SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return nil, errors.New("fail")
	}}
	if _, err := s.RemoteHome(); err == nil {
		t.Fatal("expected")
	}
}
