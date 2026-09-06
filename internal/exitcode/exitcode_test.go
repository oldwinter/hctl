package exitcode

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorMethodsNilAndValue(t *testing.T) {
	var n *Error
	if n.Error() != "" {
		t.Fatalf("nil Error() = %q", n.Error())
	}
	if n.Unwrap() != nil {
		t.Fatal("nil Unwrap")
	}
	if n.ExitCode() != Generic {
		t.Fatalf("nil ExitCode = %d", n.ExitCode())
	}
	e := &Error{Code: Usage, Err: errors.New("boom")}
	if e.Error() != "boom" {
		t.Fatalf("Error = %q", e.Error())
	}
	if e.Unwrap().Error() != "boom" {
		t.Fatal(e.Unwrap())
	}
	if e.ExitCode() != Usage {
		t.Fatalf("ExitCode = %d", e.ExitCode())
	}
	e2 := &Error{Code: Verify, Err: nil}
	if e2.Error() != "" {
		t.Fatalf("nil inner Error = %q", e2.Error())
	}
}

func TestWrapErrorfFrom(t *testing.T) {
	if Wrap(Usage, nil) != nil {
		t.Fatal("Wrap nil")
	}
	base := errors.New("x")
	w := Wrap(Parse, base)
	var e *Error
	if !errors.As(w, &e) || e.Code != Parse {
		t.Fatalf("%v", w)
	}
	// already wrapped — returned as-is
	again := Wrap(SSH, w)
	if again != w {
		t.Fatal("expected same")
	}
	ef := Errorf(Verify, "bad %s", "thing")
	if !errors.As(ef, &e) || e.Code != Verify {
		t.Fatalf("%v", ef)
	}
	if From(nil) != OK {
		t.Fatal(From(nil))
	}
	if From(errors.New("plain")) != Generic {
		t.Fatal(From(errors.New("plain")))
	}
	if From(ef) != Verify {
		t.Fatal(From(ef))
	}
	_ = fmt.Sprintf("%v", e)
}
