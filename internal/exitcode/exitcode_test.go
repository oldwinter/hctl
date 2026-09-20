package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestFromNilIsOK(t *testing.T) {
	if From(nil) != OK {
		t.Fatalf("From(nil)=%d", From(nil))
	}
}

func TestErrorfSetsCode(t *testing.T) {
	err := Errorf(Usage, "need -f")
	if From(err) != Usage {
		t.Fatalf("code=%d", From(err))
	}
	if err.Error() != "need -f" {
		t.Fatalf("msg=%q", err.Error())
	}
}

func TestWrapKeepsExistingCode(t *testing.T) {
	inner := Errorf(SSH, "down")
	wrapped := Wrap(Generic, fmt.Errorf("outer: %w", inner))
	if From(wrapped) != SSH {
		t.Fatalf("code=%d want SSH", From(wrapped))
	}
}

func TestWrapNil(t *testing.T) {
	if Wrap(Usage, nil) != nil {
		t.Fatal("Wrap(nil) should be nil")
	}
}

func TestPlainErrorIsGeneric(t *testing.T) {
	if From(errors.New("boom")) != Generic {
		t.Fatalf("plain=%d", From(errors.New("boom")))
	}
}

func TestNilReceiver(t *testing.T) {
	var err *Error
	if err.Error() != "" {
		t.Fatalf("nil Error()=%q", err.Error())
	}
	if err.Unwrap() != nil {
		t.Fatal("nil Unwrap")
	}
	if err.ExitCode() != Generic {
		t.Fatalf("nil ExitCode=%d", err.ExitCode())
	}
}
