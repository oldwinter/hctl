package adapters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/claude"
	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestDetectBinaryFSUsesRemoteCommandV(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho 'codex 9.9 fixture'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	fsys := fsx.SSH{
		Target: "user@box",
		Run:    testutil.SSHShellRunner(t),
	}
	path, ver, ok := DetectBinaryFS(fsys, []string{"fixture-missing", "codex"})
	if !ok || path != bin {
		t.Fatalf("path=%q ok=%v", path, ok)
	}
	if ver != "" {
		t.Fatalf("remote version should be empty, got %q", ver)
	}
}

func TestReadOneInstalledUsesRemoteLookPath(t *testing.T) {
	home := testutil.Testdata(t, "home-a")
	dir := t.TempDir()
	bin := filepath.Join(dir, "codex")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	called := false
	shell := testutil.SSHShellRunner(t)
	fsys := fsx.SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			if strings.Contains(testutil.SSHRemoteCommand(t, args), "command -v") {
				called = true
			}
			return shell(stdin, name, args...)
		},
	}
	snap, err := ReadOne(codex.Adapter{}, fsys, home)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected remote command -v")
	}
	if !snap.Installed || snap.InstalledPath != bin {
		t.Fatalf("installed=%v path=%q", snap.Installed, snap.InstalledPath)
	}
	if snap.Version != "" {
		t.Fatalf("remote version should be skipped, got %q", snap.Version)
	}
	if !snap.ConfigFound {
		t.Fatal("expected remote config to be readable")
	}
}

func TestDetectBinaryWithoutCommandProbesKeepsPathLookup(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "called")
	bin := filepath.Join(dir, "fixture-probe")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nprintf called > \""+marker+"\"\nprintf 'fixture version\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	path, version, ok := DetectBinaryFS(fsx.WithoutCommandProbes(fsx.Local{}), []string{"fixture-probe"})
	if !ok || path != bin || version != "" {
		t.Fatalf("path=%q version=%q ok=%v", path, version, ok)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("version subprocess ran: %v", err)
	}
}

func TestFilterDesiredFieldsSkipsClaudeProvider(t *testing.T) {
	got := FilterDesiredFields(claude.Adapter{}, []string{"model", "provider", "secret-ref"})
	if len(got) != 2 || got[0] != "model" || got[1] != "secret-ref" {
		t.Fatalf("claude fields = %#v", got)
	}
	kept := FilterDesiredFields(codex.Adapter{}, []string{"model", "provider"})
	if len(kept) != 2 {
		t.Fatalf("codex fields = %#v", kept)
	}
}
