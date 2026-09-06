package claude

import (
	"strings"
	"testing"

	"github.com/oldwinter/harnessctl/internal/secret"
	"github.com/oldwinter/harnessctl/internal/testutil"
)

func TestReadHomeA(t *testing.T) {
	snap, err := Adapter{}.Read(testutil.Testdata(t, "home-a"))
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
}
