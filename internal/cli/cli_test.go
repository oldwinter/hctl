package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
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
	if !strings.Contains(out, "commit:") || !strings.Contains(out, "built:") {
		t.Fatal(out)
	}
}

func TestGetMissingResource(t *testing.T) {
	_, err := run(t, "get")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if !strings.Contains(err.Error(), "harnesses") || !strings.Contains(err.Error(), "models") {
		t.Fatalf("err = %v", err)
	}
	if exitcode.From(err) != exitcode.Usage {
		t.Fatalf("exit = %d want %d (%v)", exitcode.From(err), exitcode.Usage, err)
	}
}

func TestGetUnknownResource(t *testing.T) {
	home := testutil.Testdata(t, "home-a")
	cfg := testutil.Testdata(t, "harnessctl.yaml")
	_, err := run(t, "--home", home, "--config", cfg, "get", "pods")
	if err == nil {
		t.Fatal("expected usage error")
	}
	if exitcode.From(err) != exitcode.Usage {
		t.Fatalf("exit = %d want %d (%v)", exitcode.From(err), exitcode.Usage, err)
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
	if !strings.Contains(out, filepath.Join(home, ".codex", "config.toml")) {
		t.Fatalf("dry-run PATH missing config path: %s", out)
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

func TestManagedDestinationGuardAppliesToDryRunAndExplicitOverride(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	backupDir := t.TempDir()
	t.Setenv("HARNESSCTL_BACKUP_DIR", backupDir)
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	writeTestFile(t, manifestPath, `{"version":1,"units":[{"dest":"~/.codex/config.toml"}]}`)
	writeTestFile(t, filepath.Join(home, ".config", "harness", "ownership.json"), fmt.Sprintf(`{"version":1,"owner":"oldwinter/dotfiles","manifest":%q}`, manifestPath))
	configPath := filepath.Join(home, ".codex", "config.toml")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = run(t, "--no-probe", "--home", home, "--config", testutil.Testdata(t, "harnessctl.yaml"), "set", "model", "codex", "blocked-model", "--dry-run")
	if err == nil || exitcode.From(err) != exitcode.Usage || !strings.Contains(err.Error(), "managed") {
		t.Fatalf("expected managed dry-run refusal, got %v", err)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("managed dry-run changed config")
	}
	assertDirectoryEmpty(t, backupDir)

	if _, err := run(t, "--no-probe", "--allow-managed", "--home", home, "--config", testutil.Testdata(t, "harnessctl.yaml"), "set", "model", "codex", "allowed-model"); err != nil {
		t.Fatal(err)
	}
	after, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), `model = "allowed-model"`) {
		t.Fatalf("override did not write model: %s", after)
	}
}

func TestApplyPreflightsWholeBatchBeforeMutation(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	backupDir := t.TempDir()
	t.Setenv("HARNESSCTL_BACKUP_DIR", backupDir)
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	writeTestFile(t, manifestPath, `{"version":1,"units":[{"dest":"~/.pi/agent/settings.json"}]}`)
	writeTestFile(t, filepath.Join(home, ".config", "harness", "ownership.json"), fmt.Sprintf(`{"version":1,"owner":"oldwinter/dotfiles","manifest":%q}`, manifestPath))
	desiredPath := filepath.Join(t.TempDir(), "desired.toml")
	writeTestFile(t, desiredPath, `apiVersion = "harnessctl/v1"
kind = "DesiredState"

[harnesses.opencode]
model = "acme/must-not-write"

[harnesses.pi]
model = "must-not-write"
`)
	opencodePath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	piPath := filepath.Join(home, ".pi", "agent", "settings.json")
	opencodeBefore, _ := os.ReadFile(opencodePath)
	piBefore, _ := os.ReadFile(piPath)

	_, err := run(t, "--no-probe", "--home", home, "--config", testutil.Testdata(t, "harnessctl.yaml"), "apply", "-f", desiredPath)
	if err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("expected managed batch refusal, got %v", err)
	}
	opencodeAfter, _ := os.ReadFile(opencodePath)
	piAfter, _ := os.ReadFile(piPath)
	if string(opencodeAfter) != string(opencodeBefore) || string(piAfter) != string(piBefore) {
		t.Fatal("apply partially mutated a preflight-rejected batch")
	}
	assertDirectoryEmpty(t, backupDir)
}

func TestManagedGuardCoversSecretOnlySync(t *testing.T) {
	src := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	dst := testutil.CopyTree(t, testutil.Testdata(t, "home-b"))
	backupDir := t.TempDir()
	t.Setenv("HARNESSCTL_BACKUP_DIR", backupDir)
	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	writeTestFile(t, manifestPath, `{"version":1,"units":[{"dest":"~/.pi/agent/auth.json"}]}`)
	writeTestFile(t, filepath.Join(dst, ".config", "harness", "ownership.json"), fmt.Sprintf(`{"version":1,"owner":"oldwinter/dotfiles","manifest":%q}`, manifestPath))
	configPath := writeLocalContexts(t, src, dst)
	authPath := filepath.Join(dst, ".pi", "agent", "auth.json")
	before, _ := os.ReadFile(authPath)

	_, err := run(t, "--no-probe", "--config", configPath, "sync", "--from", "source", "--to", "destination", "--harness", "pi", "--fields", "secret")
	if err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("expected managed secret refusal, got %v", err)
	}
	after, _ := os.ReadFile(authPath)
	if string(after) != string(before) {
		t.Fatal("managed secret sync changed auth")
	}
	assertDirectoryEmpty(t, backupDir)
}

func TestSyncSecretPreflightUsesPendingPiProvider(t *testing.T) {
	t.Run("future OAuth provider rejects entire batch", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()
		writePiHome(t, src, "provider-b", `{"provider-b":{"type":"api_key","key":"sk-test-bbb"}}`)
		writePiHome(t, dst, "provider-a", `{"provider-a":{"type":"api_key","key":"sk-test-aaa"},"provider-b":{"type":"oauth","access":"sk-test-aaa","refresh":"sk-test-bbb","expires":4102444800000}}`)
		backupDir := t.TempDir()
		t.Setenv("HARNESSCTL_BACKUP_DIR", backupDir)
		configPath := writeLocalContexts(t, src, dst)
		settingsPath := filepath.Join(dst, ".pi", "agent", "settings.json")
		authPath := filepath.Join(dst, ".pi", "agent", "auth.json")
		settingsBefore, _ := os.ReadFile(settingsPath)
		authBefore, _ := os.ReadFile(authPath)

		_, err := run(t, "--no-probe", "--config", configPath, "sync", "--from", "source", "--to", "destination", "--harness", "pi", "--fields", "provider,secret")
		if err == nil || exitcode.From(err) != exitcode.Usage {
			t.Fatalf("expected future OAuth refusal, got %v", err)
		}
		settingsAfter, _ := os.ReadFile(settingsPath)
		authAfter, _ := os.ReadFile(authPath)
		if string(settingsAfter) != string(settingsBefore) || string(authAfter) != string(authBefore) {
			t.Fatal("provider/secret batch partially mutated after refusal")
		}
		assertDirectoryEmpty(t, backupDir)
	})

	t.Run("future API key provider succeeds from OAuth current provider", func(t *testing.T) {
		src, dst := t.TempDir(), t.TempDir()
		writePiHome(t, src, "provider-b", `{"provider-b":{"type":"api_key","key":"sk-test-bbb"}}`)
		writePiHome(t, dst, "provider-a", `{"provider-a":{"type":"oauth","access":"sk-test-aaa","refresh":"sk-test-bbb","expires":4102444800000}}`)
		t.Setenv("HARNESSCTL_BACKUP_DIR", t.TempDir())
		configPath := writeLocalContexts(t, src, dst)

		if _, err := run(t, "--no-probe", "--config", configPath, "sync", "--from", "source", "--to", "destination", "--harness", "pi", "--fields", "provider,secret"); err != nil {
			t.Fatal(err)
		}
		settingsData, _ := os.ReadFile(filepath.Join(dst, ".pi", "agent", "settings.json"))
		var settings map[string]any
		if err := json.Unmarshal(settingsData, &settings); err != nil {
			t.Fatal(err)
		}
		if settings["defaultProvider"] != "provider-b" {
			t.Fatalf("provider=%v", settings["defaultProvider"])
		}
		authData, _ := os.ReadFile(filepath.Join(dst, ".pi", "agent", "auth.json"))
		var auth map[string]map[string]any
		if err := json.Unmarshal(authData, &auth); err != nil {
			t.Fatal(err)
		}
		if auth["provider-b"]["type"] != "api_key" || auth["provider-b"]["key"] != "sk-test-bbb" {
			t.Fatalf("future provider credential=%v", auth["provider-b"])
		}
	})
}

func writeLocalContexts(t *testing.T, source, destination string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeTestFile(t, path, fmt.Sprintf(`apiVersion: harnessctl/v1
kind: Config
current-context: source
contexts:
  - name: source
    context:
      kind: local
      home: %s
  - name: destination
    context:
      kind: local
      home: %s
`, source, destination))
	return path
}

func writePiHome(t *testing.T, home, provider, auth string) {
	t.Helper()
	writeTestFile(t, filepath.Join(home, ".pi", "agent", "settings.json"), fmt.Sprintf(`{"defaultProvider":%q,"defaultModel":"fixture-model"}`, provider))
	writeTestFile(t, filepath.Join(home, ".pi", "agent", "models.json"), `{"providers":{"provider-a":{"baseUrl":"https://a.example/v1"},"provider-b":{"baseUrl":"https://b.example/v1"}}}`)
	writeTestFile(t, filepath.Join(home, ".pi", "agent", "auth.json"), auth)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertDirectoryEmpty(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no backups, got %v", entries)
	}
}
