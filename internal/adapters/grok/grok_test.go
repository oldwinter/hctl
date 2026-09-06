package grok

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
	if snap.DefaultModel != "grok-4.6" {
		t.Fatalf("model = %q", snap.DefaultModel)
	}
	if snap.BaseURLHost != "api.x.ai" {
		t.Fatalf("host = %q", snap.BaseURLHost)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fp = %q", snap.SecretFingerprint)
	}
	if strings.Contains(snap.String(), "sk-test-aaa") {
		t.Fatalf("leaked: %s", snap.String())
	}
}
