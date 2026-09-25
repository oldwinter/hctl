package hermes

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
	if snap.DefaultModel != "anthropic/claude-opus-4.6" {
		t.Fatalf("model = %q", snap.DefaultModel)
	}
	if snap.Provider != "custom" {
		t.Fatalf("provider = %q", snap.Provider)
	}
	if snap.BaseURLHost != "openrouter.ai" {
		t.Fatalf("host = %q", snap.BaseURLHost)
	}
	if snap.SecretRef != "OPENROUTER_API_KEY" {
		t.Fatalf("ref = %q", snap.SecretRef)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fp = %q", snap.SecretFingerprint)
	}
	if strings.Contains(snap.String(), "sk-test-aaa") {
		t.Fatalf("leaked: %s", snap.String())
	}
}

func TestReadCurrentCustomProviderSchema(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data := `model:
  default: gpt-5.6-sol
  provider: custom:sub2api
providers:
  sub2api:
    base_url: https://gateway.example/v1
    key_env: HERMES_CUSTOM_SUB2API_API_KEY
custom_providers:
  - name: fixture-gateway
    base_url: https://gateway.example
    api_key: sk-test-aaa
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "gpt-5.6-sol" || snap.Provider != "custom:sub2api" || snap.BaseURLHost != "gateway.example" {
		t.Fatalf("snapshot=%+v", snap)
	}
	if snap.SecretRef != "HERMES_CUSTOM_SUB2API_API_KEY" || snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("snapshot=%+v", snap)
	}
	if err := (Adapter{}).ValidateSecretWrite(fsx.Local{}, home, ""); err == nil {
		t.Fatal("expected inline custom provider secret write refusal")
	}
}

func TestCustomProviderFallbackMatchesFullEndpointInsteadOfHost(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data := `model:
  default: gpt-5.6-sol
  provider: custom:gateway
providers:
  gateway:
    api: https://gateway.example/team-a/v1
custom_providers:
  - name: team-b
    base_url: https://gateway.example/team-b
    api_key: sk-test-bbb
  - name: team-a
    base_url: https://gateway.example/team-a
    api_key: sk-test-aaa
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("selected wrong same-host credential: %+v", snap)
	}
	_, value, err := (Adapter{}).PeekSecret(fsx.Local{}, home)
	if err != nil || value != "sk-test-aaa" {
		t.Fatalf("peek selected %q, err=%v", value, err)
	}
}

func TestCustomProviderFallbackRejectsAmbiguousEndpoint(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data := `model:
  default: gpt-5.6-sol
  provider: custom:gateway
providers:
  gateway:
    base_url: https://gateway.example/team/v1
custom_providers:
  - name: first
    base_url: https://gateway.example/team
    api_key: sk-test-aaa
  - name: second
    base_url: https://gateway.example/team/v1
    api_key: sk-test-bbb
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.SecretFingerprint != "" || len(snap.Notes) == 0 {
		t.Fatalf("ambiguous credential should stay unknown: %+v", snap)
	}
	_, _, err = (Adapter{}).PeekSecret(fsx.Local{}, home)
	if err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("expected ambiguous secret refusal, got %v", err)
	}
}

func TestEmptyModelOnboarding(t *testing.T) {
	snap, err := Adapter{}.Read(fsx.Local{}, testutil.Testdata(t, "home-b"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "" {
		t.Fatalf("model = %q", snap.DefaultModel)
	}
	if len(snap.Notes) == 0 {
		t.Fatal("expected onboarding note")
	}
}

func TestValidateDesiredRejectsUnknownProvider(t *testing.T) {
	err := (Adapter{}).ValidateDesired(fsx.Local{}, testutil.Testdata(t, "home-a"), model.Desired{Provider: "custom:openrouter"})
	if err == nil || !strings.Contains(err.Error(), "providers") {
		t.Fatalf("err=%v", err)
	}
}

func writeConfig(t *testing.T, home, data string) string {
	t.Helper()
	path := filepath.Join(home, ".hermes", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadEndpointReturnsFullProviderURL(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, `model:
  default: gpt-5.6-sol
  provider: custom:gateway
providers:
  gateway:
    base_url: https://gateway.example/team/v1
`)
	endpoint, err := (Adapter{}).ReadEndpoint(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if endpoint != "https://gateway.example/team/v1" {
		t.Fatalf("endpoint = %q", endpoint)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.BaseURLHost != "gateway.example" {
		t.Fatalf("snapshot should still expose only the host: %+v", snap)
	}
}

func TestWriteFieldsBaseURLRetargetsExistingProvider(t *testing.T) {
	home := t.TempDir()
	path := writeConfig(t, home, `model:
  default: gpt-5.6-sol
  provider: custom:gateway
providers:
  gateway:
    base_url: https://old.example/v1
`)
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{BaseURL: "https://new.example/team/v2"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "base_url: https://new.example/team/v2") {
		t.Fatalf("config: %s", data)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.BaseURLHost != "new.example" {
		t.Fatalf("host after write = %q", snap.BaseURLHost)
	}
}

func TestValidateDesiredRejectsBaseURLForMissingProvider(t *testing.T) {
	home := t.TempDir()
	writeConfig(t, home, `model:
  default: gpt-5.6-sol
  provider: custom:gateway
providers:
  gateway:
    base_url: https://gateway.example/v1
`)
	err := (Adapter{}).ValidateDesired(fsx.Local{}, home, model.Desired{Provider: "other", BaseURL: "https://new.example"})
	if err == nil || !strings.Contains(err.Error(), "providers") {
		t.Fatalf("err=%v", err)
	}
	err = (Adapter{}).ValidateDesired(fsx.Local{}, t.TempDir(), model.Desired{BaseURL: "https://new.example"})
	if err == nil || !strings.Contains(err.Error(), "provider") {
		t.Fatalf("err=%v", err)
	}
}
