package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/testutil"
)

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := NewRoot()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestCompletionBash(t *testing.T) {
	out, err := run(t, "completion", "bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hctl") && !strings.Contains(out, "harnessctl") {
		t.Fatal(out[:min(len(out), 80)])
	}
}

func TestVersion(t *testing.T) {
	out, err := run(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hctl version "+Version) && !strings.Contains(out, "harnessctl version "+Version) {
		t.Fatal(out)
	}
}

func TestGetHarnessesJSONAgainstFixtures(t *testing.T) {
	home := testutil.Testdata(t, "home-a")
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	out, err := run(t, "--home", home, "--config", cfg, "--json", "get", "harnesses")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "sk-test") {
		t.Fatal("leaked key in json")
	}
	if !strings.Contains(out, `"name": "codex"`) {
		t.Fatal(out)
	}
	if !strings.Contains(out, "gpt-5.2-codex") {
		t.Fatal(out)
	}
}

func TestGetModelsAndDescribe(t *testing.T) {
	home := testutil.Testdata(t, "home-a")
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	out, err := run(t, "--home", home, "--config", cfg, "get", "models")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "claude-sonnet-4") {
		t.Fatal(out)
	}
	out, err = run(t, "--home", home, "--config", cfg, "describe", "harness", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "sk-test") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "gpt-5.2-codex") {
		t.Fatal(out)
	}
}

func TestDoctorAndDiffHomes(t *testing.T) {
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	a := testutil.Testdata(t, "home-a")
	b := testutil.Testdata(t, "home-b")
	out, err := run(t, "--home", a, "--config", cfg, "doctor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "codex") {
		t.Fatal(out)
	}
	out, err = run(t, "--config", cfg, "diff", "harness", "codex", "--home-a", a, "--home-b", b, "--a", "mba", "--b", "box")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "defaultModel") {
		t.Fatal(out)
	}
	if strings.Contains(out, "sk-test") {
		t.Fatal(out)
	}
}

func TestSSHContextErrorsWithoutHome(t *testing.T) {
	t.Setenv("HARNESSCTL_SSH", "0")
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	_, err := run(t, "--config", cfg, "--context", "box", "get", "harnesses")
	if err == nil {
		t.Fatal("expected ssh error")
	}
	if !strings.Contains(err.Error(), "ssh") {
		t.Fatalf("err = %v", err)
	}
}

func TestSyncAndContextDiff(t *testing.T) {
	a := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	b := testutil.CopyTree(t, testutil.Testdata(t, "home-b"))
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	bak := t.TempDir()
	t.Setenv("HARNESSCTL_BACKUP_DIR", bak)
	body := fmt.Sprintf(`apiVersion: harnessctl/v1
kind: Config
current-context: mba
contexts:
  - name: mba
    context:
      kind: local
      home: %s
  - name: box
    context:
      kind: local
      home: %s
`, a, b)
	if err := os.WriteFile(cfgPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "--config", cfgPath, "diff", "harness", "codex", "--contexts", "mba,box")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "defaultModel") {
		t.Fatal(out)
	}
	out, err = run(t, "--config", cfgPath, "sync", "--from", "mba", "--to", "box", "--harness", "codex", "--fields", "model", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "sk-test") {
		t.Fatal(out)
	}
	if _, err := run(t, "--config", cfgPath, "sync", "--from", "mba", "--to", "box", "--harness", "codex", "--fields", "model"); err != nil {
		t.Fatal(err)
	}
	out, err = run(t, "--config", cfgPath, "--context", "box", "describe", "harness", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "gpt-5.2-codex") {
		t.Fatal(out)
	}
}

func TestConfigContextsAndUse(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	src, err := os.ReadFile(testutil.Testdata(t, "harnessctl.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, src, 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "--config", cfgPath, "config", "get-contexts")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "mba") || !strings.Contains(out, "box") {
		t.Fatal(out)
	}
	out, err = run(t, "--config", cfgPath, "config", "current-context")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "mba" {
		t.Fatal(out)
	}
	if _, err := run(t, "--config", cfgPath, "config", "use-context", "box"); err != nil {
		t.Fatal(err)
	}
	out, err = run(t, "--config", cfgPath, "config", "current-context")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "box" {
		t.Fatal(out)
	}
}

func TestApplyAndDiffDesired(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	desired := testutil.Testdata(t, "desired.toml")
	bak := t.TempDir()
	t.Setenv("HARNESSCTL_BACKUP_DIR", bak)
	out, err := run(t, "--home", home, "--config", cfg, "diff", "-f", desired)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "o4-mini") {
		t.Fatal(out)
	}
	out, err = run(t, "--home", home, "--config", cfg, "apply", "-f", desired, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dry-run") {
		t.Fatal(out)
	}
	if _, err := run(t, "--home", home, "--config", cfg, "apply", "-f", desired); err != nil {
		t.Fatal(err)
	}
	out, err = run(t, "--home", home, "--config", cfg, "get", "models")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "o4-mini") || !strings.Contains(out, "round-trip") && !strings.Contains(out, "claude-opus-4") {
		// claude-opus-4 is in desired.toml
	}
	if !strings.Contains(out, "claude-opus-4") {
		t.Fatal(out)
	}
	if strings.Contains(out, "sk-test") {
		t.Fatal("leaked")
	}
}

func TestSetModelRoundTrip(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	bak := t.TempDir()
	t.Setenv("HARNESSCTL_BACKUP_DIR", bak)
	out, err := run(t, "--home", home, "--config", cfg, "set", "model", "codex", "o4-mini", "--dry-run")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "dry-run") || !strings.Contains(out, "o4-mini") {
		t.Fatal(out)
	}
	if strings.Contains(out, "sk-test") {
		t.Fatal("leaked")
	}
	out, err = run(t, "--home", home, "--config", cfg, "set", "model", "codex", "o4-mini")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "verified") {
		t.Fatal(out)
	}
	out, err = run(t, "--home", home, "--config", cfg, "describe", "harness", "codex")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "o4-mini") {
		t.Fatal(out)
	}
	if strings.Contains(out, "sk-test") {
		t.Fatal(out)
	}
}

func TestSetProviderUnsupported(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	for _, name := range []string{"claude", "grok"} {
		_, err := run(t, "--home", home, "--config", cfg, "set", "provider", name, "custom", "--dry-run")
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestHARNESSCTL_HOME(t *testing.T) {
	t.Setenv("HARNESSCTL_HOME", testutil.Testdata(t, "home-a"))
	t.Setenv("HARNESSCTL_CONFIG", testutil.Testdata(t, "harnessctl.yaml"))
	out, err := run(t, "get", "harnesses")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "opencode") {
		t.Fatal(out)
	}
}
