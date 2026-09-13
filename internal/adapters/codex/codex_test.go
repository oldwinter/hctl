package codex

import (
	"strings"
	"testing"

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
