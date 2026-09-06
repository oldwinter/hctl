package hermes

import (
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/secret"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestReadHomeA(t *testing.T) {
	snap, err := Adapter{}.Read(testutil.Testdata(t, "home-a"))
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

func TestEmptyModelOnboarding(t *testing.T) {
	snap, err := Adapter{}.Read(testutil.Testdata(t, "home-b"))
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
