package adapters

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestDetectBinaryFSUsesRemoteCommandV(t *testing.T) {
	fsys := fsx.SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			if name != "ssh" {
				t.Fatalf("name=%s", name)
			}
			script := args[len(args)-1]
			if strings.Contains(script, "command -v") && strings.Contains(script, "codex") {
				return []byte("/usr/bin/codex\n"), nil
			}
			if strings.Contains(script, "command -v") {
				return nil, fmt.Errorf("exit status 1")
			}
			if strings.Contains(script, "--version") {
				t.Fatal("remote version probe must not run")
			}
			return nil, fmt.Errorf("unexpected remote cmd: %s", script)
		},
	}
	path, ver, ok := DetectBinaryFS(fsys, []string{"missing", "codex"})
	if !ok || path != "/usr/bin/codex" {
		t.Fatalf("path=%q ok=%v", path, ok)
	}
	if ver != "" {
		t.Fatalf("remote version should be empty, got %q", ver)
	}
}

func TestReadOneInstalledUsesRemoteLookPath(t *testing.T) {
	home := testutil.Testdata(t, "home-a")
	called := false
	fsys := fsx.SSH{
		Target: "user@box",
		Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
			script := args[len(args)-1]
			if strings.Contains(script, "command -v") {
				called = true
				if strings.Contains(script, "codex") {
					return []byte("/opt/box/bin/codex\n"), nil
				}
				return nil, fmt.Errorf("exit status 1")
			}
			cmd := exec.Command("sh", "-c", script)
			if stdin != nil {
				cmd.Stdin = bytes.NewReader(stdin)
			}
			out, err := cmd.Output()
			return out, err
		},
	}
	snap, err := ReadOne(codex.Adapter{}, fsys, home)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected remote command -v")
	}
	if !snap.Installed || snap.InstalledPath != "/opt/box/bin/codex" {
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
