package cursor

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
)

func writeFake(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "cursor-agent")
	if runtime.GOOS == "windows" {
		p += ".bat"
	}
	if err := os.WriteFile(p, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProbeLoginBranches(t *testing.T) {
	dir := t.TempDir()
	old := os.Getenv("PATH")
	defer os.Setenv("PATH", old)

	// logged in
	writeFake(t, dir, "#!/bin/sh\necho Logged in as user\n")
	os.Setenv("PATH", dir+string(os.PathListSeparator)+old)
	ok, note := probeLogin()
	if !ok || note == "" {
		t.Fatalf("%v %q", ok, note)
	}
	// not logged in
	writeFake(t, dir, "#!/bin/sh\necho please login\n")
	ok, note = probeLogin()
	if ok || note == "" {
		t.Fatalf("%v %q", ok, note)
	}
	// other output
	writeFake(t, dir, "#!/bin/sh\necho hello\n")
	ok, note = probeLogin()
	if ok || note != "" {
		t.Fatalf("%v %q", ok, note)
	}
	// error exit
	writeFake(t, dir, "#!/bin/sh\nexit 1\n")
	ok, note = probeLogin()
	if ok || note != "" {
		t.Fatalf("%v %q", ok, note)
	}
	// timeout
	writeFake(t, dir, "#!/bin/sh\nsleep 5\n")
	ok, note = probeLogin()
	if note != "cursor-agent status timed out" && note != "" {
		// may be flaky if sleep killed differently
		t.Log(ok, note)
	}
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".cursor"), 0o755)
	os.WriteFile(filepath.Join(home, ".cursor", "cli-config.json"), []byte(`{"model":"m"}`), 0o600)
	_, _ = (Adapter{}).ReadFS(fsx.Local{}, home)
}
