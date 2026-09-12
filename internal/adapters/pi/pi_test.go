package pi

import (
	"encoding/json"
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
	if snap.DefaultModel != "gpt-5.6-sol" {
		t.Fatalf("model=%q", snap.DefaultModel)
	}
	if snap.Provider != "sub2api" {
		t.Fatalf("provider=%q", snap.Provider)
	}
	if snap.BaseURLHost != "api.openai.com" {
		t.Fatalf("host=%q", snap.BaseURLHost)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fp=%q", snap.SecretFingerprint)
	}
	if strings.Contains(snap.String(), "sk-test") {
		t.Fatal(snap.String())
	}
}

func TestProviderScopedAuthTakesPrecedenceOverModelsJSON(t *testing.T) {
	snap, err := Adapter{}.Read(fsx.Local{}, testutil.Testdata(t, "home-a"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.SecretFingerprint == secret.Fingerprint("sk-test-bbb") {
		t.Fatal("models.json key incorrectly won over provider-scoped auth.json")
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fingerprint=%q", snap.SecretFingerprint)
	}
}

func TestProviderScopedAuthEnvMappingAndUnresolvedPrecedence(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	authPath := filepath.Join(home, ".pi", "agent", "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"sub2api":{"type":"api_key","key":"$PI_FIXTURE_KEY","env":{"PI_FIXTURE_KEY":"sk-test-aaa"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.SecretRef != "PI_FIXTURE_KEY" || snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("snapshot=%+v", snap)
	}
	ref, value, err := (Adapter{}).PeekSecret(fsx.Local{}, home)
	if err != nil || ref != "PI_FIXTURE_KEY" || value != "sk-test-aaa" {
		t.Fatalf("ref=%q value=%q err=%v", ref, value, err)
	}

	if err := os.WriteFile(authPath, []byte(`{"sub2api":{"type":"api_key"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err = (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.SecretFingerprint != "" || snap.SecretRef != "" {
		t.Fatalf("unresolved auth entry fell through to models.json: %+v", snap)
	}
}

func TestWriteFieldsUsesCanonicalKeysAndPreservesUnrelatedSettings(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	paths, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Model: "fixture-next", Provider: "other"})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || filepath.Base(paths[0]) != "settings.json" {
		t.Fatalf("paths=%v", paths)
	}
	data, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["defaultModel"] != "fixture-next" || raw["defaultProvider"] != "other" || raw["theme"] != "dark" {
		t.Fatalf("settings=%v", raw)
	}
	if _, ok := raw["model"]; ok {
		t.Fatalf("legacy model key written: %v", raw)
	}
	if _, ok := raw["provider"]; ok {
		t.Fatalf("legacy provider key written: %v", raw)
	}
}

func TestSecretRefWritesSelectedProviderCredential(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "PI_FIXTURE_KEY"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".pi", "agent", "auth.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw["sub2api"]["type"] != "api_key" || raw["sub2api"]["key"] != "$PI_FIXTURE_KEY" {
		t.Fatalf("auth=%v", raw)
	}
	if raw["unused"]["key"] != "sk-test-bbb" {
		t.Fatal("unrelated provider credential changed")
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil || snap.SecretRef != "PI_FIXTURE_KEY" {
		t.Fatalf("snapshot=%+v err=%v", snap, err)
	}
}

func TestPeekSecretRejectsCommandExpressionWithoutExecutingIt(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	authPath := filepath.Join(home, ".pi", "agent", "auth.json")
	if err := os.WriteFile(authPath, []byte(`{"sub2api":{"type":"api_key","key":"!printf should-not-run"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := (Adapter{}).PeekSecret(fsx.Local{}, home)
	if err == nil || exitcode.From(err) != exitcode.Usage || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("expected unsupported command reference, got %v", err)
	}
}

func TestSecretRefRefusesSelectedOAuthCredentialWithoutChangingAuth(t *testing.T) {
	for _, tc := range []struct {
		name    string
		auth    string
		desired model.Desired
	}{
		{
			name:    "current provider",
			auth:    `{"sub2api":{"type":"oauth","access":"sk-test-aaa","refresh":"sk-test-bbb","expires":4102444800000}}`,
			desired: model.Desired{SecretRef: "PI_FIXTURE_KEY"},
		},
		{
			name:    "provider override",
			auth:    `{"sub2api":{"type":"api_key","key":"sk-test-aaa"},"oauth-target":{"type":"oauth","access":"sk-test-aaa","refresh":"sk-test-bbb","expires":4102444800000}}`,
			desired: model.Desired{Provider: "oauth-target", SecretRef: "PI_FIXTURE_KEY"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
			authPath := filepath.Join(home, ".pi", "agent", "auth.json")
			before := []byte(tc.auth)
			if err := os.WriteFile(authPath, before, 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := (Adapter{}).WriteFields(fsx.Local{}, home, tc.desired)
			if err == nil || exitcode.From(err) != exitcode.Usage {
				t.Fatalf("expected OAuth usage refusal, got %v", err)
			}
			after, readErr := os.ReadFile(authPath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(after) != string(before) {
				t.Fatalf("OAuth credential changed: before=%q after=%q", before, after)
			}
		})
	}
}
