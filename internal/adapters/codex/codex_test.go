package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestReadHomeA(t *testing.T) {
	snap, err := Adapter{}.Read(fsx.Local{}, testutil.Testdata(t, "home-a"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "gpt-5.2-codex" {
		t.Fatalf("model = %q", snap.DefaultModel)
	}
	if snap.Provider != "custom" {
		t.Fatalf("provider = %q", snap.Provider)
	}
	if snap.BaseURLHost != "api.openai.com" {
		t.Fatalf("host = %q", snap.BaseURLHost)
	}
	if snap.Effort != "high" {
		t.Fatalf("effort = %q", snap.Effort)
	}
	want := secret.Fingerprint("sk-test-aaa")
	if snap.SecretFingerprint != want {
		t.Fatalf("fp = %q want %q", snap.SecretFingerprint, want)
	}
	if strings.Contains(snap.String(), "sk-test-aaa") {
		t.Fatalf("String leaked secret: %s", snap.String())
	}
}

func TestValidateDesiredRejectsUnknownProvider(t *testing.T) {
	err := (Adapter{}).ValidateDesired(fsx.Local{}, testutil.Testdata(t, "home-a"), model.Desired{Provider: "openai"})
	if err == nil || !strings.Contains(err.Error(), "model_providers") {
		t.Fatalf("err=%v", err)
	}
}

func TestMissingConfig(t *testing.T) {
	snap, err := Adapter{}.Read(fsx.Local{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if snap.ConfigFound {
		t.Fatal("expected missing")
	}
}

func writeCodexConfig(t *testing.T, home, data string) string {
	t.Helper()
	path := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSecretWritesFailClosedWithoutSelectedProvider(t *testing.T) {
	home := t.TempDir()
	path := writeCodexConfig(t, home, `model = "o4-mini"
`)
	before, _ := os.ReadFile(path)

	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "SRC_ENV"}); err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("secretRef write err=%v", err)
	}
	if err := (Adapter{}).WriteSecret(fsx.Local{}, home, "SRC_ENV", "sk-test-bbb"); err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("bearer write err=%v", err)
	}
	if err := (Adapter{}).ValidateSecretWrite(fsx.Local{}, home, model.Desired{}); err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("validate err=%v", err)
	}
	if err := (Adapter{}).ValidateSecretWrite(fsx.Local{}, home, model.Desired{Provider: "custom"}); err != nil {
		t.Fatalf("pending provider should satisfy the selector: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("selector-less write mutated config:\n%s", after)
	}
}

func TestSecretWritesTargetConfiguredProvider(t *testing.T) {
	home := t.TempDir()
	path := writeCodexConfig(t, home, `model = "o4-mini"
model_provider = "openai"
`)
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "SRC_ENV"}); err != nil {
		t.Fatal(err)
	}
	if err := (Adapter{}).WriteSecret(fsx.Local{}, home, "", "sk-test-bbb"); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.SecretRef != "SRC_ENV" || snap.SecretFingerprint != secret.Fingerprint("sk-test-bbb") {
		t.Fatalf("snapshot=%+v", snap)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "[model_providers.openai]") {
		t.Fatalf("secret fields should live under the configured provider:\n%s", data)
	}
	if strings.Contains(string(data), "[model_providers.custom]") {
		t.Fatalf("write remapped to a dead custom table:\n%s", data)
	}
}

func TestSecretRefWriteWithPendingProvider(t *testing.T) {
	home := t.TempDir()
	writeCodexConfig(t, home, `model = "o4-mini"

[model_providers.custom]
name = "Custom Gateway"
base_url = "https://api.openai.com/v1"
`)
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Provider: "custom", SecretRef: "SRC_ENV"}); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Provider != "custom" || snap.SecretRef != "SRC_ENV" {
		t.Fatalf("snapshot=%+v", snap)
	}
}
