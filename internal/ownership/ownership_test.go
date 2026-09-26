package ownership

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestPointerBlocksManagedPathAndAllowsExplicitOverride(t *testing.T) {
	home := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	writeFile(t, manifestPath, `{"version":1,"units":[{"dest":"~/.pi/agent/settings.json"}]}`)
	writePointer(t, home, manifestPath)
	candidate := filepath.Join(home, ".pi", "agent", "settings.json")

	err := Check(fsx.Local{}, home, []string{candidate}, Options{})
	if err == nil || exitcode.From(err) != exitcode.Usage || !strings.Contains(err.Error(), "managed") {
		t.Fatalf("expected managed usage error, got %v", err)
	}
	if err := Check(fsx.Local{}, home, []string{candidate}, Options{AllowManaged: true}); err != nil {
		t.Fatalf("allow managed: %v", err)
	}
}

func TestInvalidPointerFailsClosedEvenWithOverride(t *testing.T) {
	cases := map[string]string{
		"malformed": `{"version":`,
		"version":   `{"version":2,"owner":"oldwinter/dotfiles","manifest":"/tmp/manifest.json"}`,
		"owner":     `{"version":1,"owner":"someone/else","manifest":"/tmp/manifest.json"}`,
		"relative":  `{"version":1,"owner":"oldwinter/dotfiles","manifest":"relative.json"}`,
		"missing":   `{"version":1,"owner":"oldwinter/dotfiles","manifest":"/missing/manifest.json"}`,
	}
	for name, pointer := range cases {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			writeFile(t, filepath.Join(home, pointerPath), pointer)
			err := Check(fsx.Local{}, home, nil, Options{AllowManaged: true})
			if err == nil || exitcode.From(err) != exitcode.Usage {
				t.Fatalf("expected fail-closed usage error, got %v", err)
			}
			if strings.Contains(err.Error(), pointer) {
				t.Fatal("error included raw pointer content")
			}
		})
	}
}

func TestInvalidExplicitManifestFailsClosedEvenWithOverride(t *testing.T) {
	cases := map[string]string{
		"malformed": `{"version":`,
		"version":   `{"version":2,"units":[]}`,
		"units":     `{"version":1}`,
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			manifestPath := filepath.Join(t.TempDir(), "manifest.json")
			writeFile(t, manifestPath, content)
			err := Check(fsx.Local{}, t.TempDir(), nil, Options{Manifest: manifestPath, AllowManaged: true})
			if err == nil || exitcode.From(err) != exitcode.Usage {
				t.Fatalf("expected fail-closed usage error, got %v", err)
			}
			if strings.Contains(err.Error(), content) {
				t.Fatal("error included raw manifest content")
			}
		})
	}

	missing := filepath.Join(t.TempDir(), "missing.json")
	if err := Check(fsx.Local{}, t.TempDir(), nil, Options{Manifest: missing}); err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("missing explicit manifest: expected usage error, got %v", err)
	}
}

func TestValidateAbsoluteAcceptsCanonicalWindowsPaths(t *testing.T) {
	for _, value := range []string{`C:\Users\fixture\dotfiles\harness\manifest.json`, `D:/Users/fixture/dotfiles/harness/manifest.json`} {
		if err := validateAbsolute(value); err != nil {
			t.Fatalf("%q: %v", value, err)
		}
	}
	for _, value := range []string{`C:relative\manifest.json`, `C:\Users\..\manifest.json`, `C:\Users\\manifest.json`} {
		if err := validateAbsolute(value); err == nil {
			t.Fatalf("%q: expected validation error", value)
		}
	}
}

func TestManifestRejectsNonCanonicalDestinations(t *testing.T) {
	for _, destination := range []string{".pi/settings.json", "~/", "~//absolute", "~/../escape", "~/.pi/./settings.json", `~\settings.json`} {
		t.Run(strings.NewReplacer("/", "_", "\\", "_").Replace(destination), func(t *testing.T) {
			home := t.TempDir()
			manifestPath := filepath.Join(t.TempDir(), "manifest.json")
			writeFile(t, manifestPath, fmt.Sprintf(`{"version":1,"units":[{"dest":%q}]}`, destination))
			err := Check(fsx.Local{}, home, nil, Options{Manifest: manifestPath})
			if err == nil || exitcode.From(err) != exitcode.Usage {
				t.Fatalf("destination %q: expected usage error, got %v", destination, err)
			}
		})
	}
}

func TestConventionalManifestFallbackAndStandaloneMode(t *testing.T) {
	home := t.TempDir()
	candidate := filepath.Join(home, ".codex", "config.toml")
	if err := Check(fsx.Local{}, home, []string{candidate}, Options{}); err != nil {
		t.Fatalf("standalone target: %v", err)
	}
	writeFile(t, filepath.Join(home, "dotfiles", "harness", "manifest.json"), `{"version":1,"units":[{"dest":"~/.codex/config.toml"}]}`)
	if err := Check(fsx.Local{}, home, []string{candidate}, Options{}); err == nil {
		t.Fatal("expected conventional manifest to guard destination")
	}
}

func TestOwnershipCheckUsesSSHFilesystem(t *testing.T) {
	home := t.TempDir()
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	writeFile(t, manifestPath, `{"version":1,"units":[{"dest":"~/.pi/agent/auth.json"}]}`)
	writePointer(t, home, manifestPath)
	remote := fsx.SSH{
		Target: "fixture@host",
		Run:    testutil.SSHShellRunner(t),
	}
	err := Check(remote, home, []string{remote.Join(home, ".pi", "agent", "auth.json")}, Options{})
	if err == nil || !strings.Contains(err.Error(), "managed") {
		t.Fatalf("expected SSH-managed refusal, got %v", err)
	}
}

func writePointer(t *testing.T, home, manifestPath string) {
	t.Helper()
	writeFile(t, filepath.Join(home, pointerPath), fmt.Sprintf(`{"version":1,"owner":"oldwinter/dotfiles","manifest":%q}`, manifestPath))
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
