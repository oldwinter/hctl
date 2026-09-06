package pi

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
	if snap.DefaultModel != "openai/gpt-5" {
		t.Fatalf("model=%q", snap.DefaultModel)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fp=%q", snap.SecretFingerprint)
	}
	if strings.Contains(snap.String(), "sk-test") {
		t.Fatal(snap.String())
	}
}
