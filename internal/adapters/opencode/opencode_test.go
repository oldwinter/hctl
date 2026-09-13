package opencode

import (
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestReadHomeAJSONC(t *testing.T) {
	snap, err := Adapter{}.Read(fsx.Local{}, testutil.Testdata(t, "home-a"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "acme/gpt-5" {
		t.Fatalf("model = %q", snap.DefaultModel)
	}
	if snap.Provider != "acme" {
		t.Fatalf("provider = %q", snap.Provider)
	}
	if snap.BaseURLHost != "api.acme.example" {
		t.Fatalf("host = %q", snap.BaseURLHost)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fp = %q", snap.SecretFingerprint)
	}
	if strings.Contains(snap.String(), "sk-test-aaa") {
		t.Fatalf("leaked: %s", snap.String())
	}
}

func TestReadHomeBEnvRefAndProvidersKey(t *testing.T) {
	snap, err := Adapter{}.Read(fsx.Local{}, testutil.Testdata(t, "home-b"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "acme/gpt-4.1" {
		t.Fatalf("model = %q", snap.DefaultModel)
	}
	if snap.SecretRef != "OPENCODE_API_KEY" {
		t.Fatalf("ref = %q", snap.SecretRef)
	}
	if snap.SecretPresent {
		t.Fatal("env ref is not an inline secret")
	}
	if snap.BaseURLHost != "llm.box.example" {
		t.Fatalf("host = %q", snap.BaseURLHost)
	}
}

func TestReadHomeANotesJSONCCommentDrop(t *testing.T) {
	snap, err := Adapter{}.Read(fsx.Local{}, testutil.Testdata(t, "home-a"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range snap.Notes {
		if strings.Contains(n, "JSONC") {
			found = true
		}
	}
	if !found {
		t.Fatalf("notes=%v", snap.Notes)
	}
}

func TestValidateDesiredRejectsProviderSlash(t *testing.T) {
	home := testutil.Testdata(t, "home-a")
	err := (Adapter{}).ValidateDesired(fsx.Local{}, home, model.Desired{Provider: "acme/gpt-4.1"})
	if err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("err=%v", err)
	}
}

func TestWriteBareModelKeepsProviderPrefix(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Model: "gpt-4.1"}); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "acme/gpt-4.1" || snap.Provider != "acme" || snap.BaseURLHost != "api.acme.example" {
		t.Fatalf("snapshot=%+v", snap)
	}
}
