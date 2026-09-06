package exitcode

import (
	"errors"
	"fmt"
)

// Stable process exit codes for harnessctl.
const (
	OK      = 0
	Generic = 1
	Usage   = 2
	Verify  = 3
	SSH     = 4
	Parse   = 5
)

// Error carries a stable exit code.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *Error) ExitCode() int {
	if e == nil {
		return Generic
	}
	return e.Code
}

func Wrap(code int, err error) error {
	if err == nil {
		return nil
	}
	var existing *Error
	if errors.As(err, &existing) {
		return err
	}
	return &Error{Code: code, Err: err}
}

func Errorf(code int, format string, args ...any) error {
	return Wrap(code, fmt.Errorf(format, args...))
}

// From returns a process code for any error.
func From(err error) int {
	if err == nil {
		return OK
	}
	var e *Error
	if errors.As(err, &e) {
		return e.ExitCode()
	}
	return Generic
}
