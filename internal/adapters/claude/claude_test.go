package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/secret"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestReadHomeA(t *testing.T) {
	snap, err := (Adapter{}).Read(testutil.Testdata(t, "home-a"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "claude-sonnet-4" {
		t.Fatalf("model = %q", snap.DefaultModel)
	}
	if snap.BaseURLHost != "api.anthropic.com" {
		t.Fatalf("host = %q", snap.BaseURLHost)
	}
	if snap.Aliases["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "claude-sonnet-4" {
		t.Fatalf("aliases = %#v", snap.Aliases)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fp = %q", snap.SecretFingerprint)
	}
	if strings.Contains(snap.String(), "sk-test") {
		t.Fatalf("leaked: %s", snap.String())
	}
	for _, n := range snap.Notes {
		if strings.Contains(n, "onboarding") {
			t.Fatalf("home-a should be onboarded: %v", snap.Notes)
		}
	}
}

func TestOnboardingJSONIncomplete(t *testing.T) {
	snap, err := (Adapter{}).Read(testutil.Testdata(t, "home-theme"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range snap.Notes {
		if strings.Contains(n, "onboarding") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected onboarding note, got %v", snap.Notes)
	}
}

func TestOnboardingJSONMissingFile(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"model":"claude-sonnet-4","env":{"ANTHROPIC_AUTH_TOKEN":"sk-test-aaa"}}`
	if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(home)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range snap.Notes {
		if strings.Contains(n, "missing ~/.claude.json") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected missing .claude.json note, got %v", snap.Notes)
	}
}
