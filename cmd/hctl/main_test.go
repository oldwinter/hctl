package main

import (
	"bytes"
	"os"
	"testing"
)

func TestRunOKAndErr(t *testing.T) {
	var buf bytes.Buffer
	if code := run([]string{"version"}, &buf); code != 0 {
		t.Fatalf("code=%d %s", code, buf.String())
	}
	buf.Reset()
	if code := run([]string{"bogus-command-xyz"}, &buf); code == 0 {
		t.Fatal("expected error")
	}
}

func TestMainCoverage(t *testing.T) {
	oldArgs := os.Args
	oldExit := osExit
	defer func() {
		os.Args = oldArgs
		osExit = oldExit
	}()
	var got int
	osExit = func(code int) {
		got = code
		panic("osExit")
	}
	os.Args = []string{"hctl", "version"}
	func() {
		defer func() { _ = recover() }()
		main()
	}()
	if got != 0 {
		t.Fatalf("got=%d", got)
	}
	os.Args = []string{"hctl", "nope-nope"}
	func() {
		defer func() { _ = recover() }()
		main()
	}()
	if got == 0 {
		t.Fatal("expected nonzero")
	}
}
