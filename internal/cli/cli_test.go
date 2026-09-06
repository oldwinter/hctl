package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/harnessctl/internal/testutil"
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

func TestVersion(t *testing.T) {
	out, err := run(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "harnessctl version "+Version) {
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
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	_, err := run(t, "--config", cfg, "--context", "box", "get", "harnesses")
	if err == nil {
		t.Fatal("expected ssh stub error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("err = %v", err)
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
