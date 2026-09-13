package droid

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
	if snap.DefaultModel != "custom:gpt-5-0" {
		t.Fatalf("%q", snap.DefaultModel)
	}
	if snap.Provider != "openai" || snap.BaseURLHost != "api.openai.com" {
		t.Fatalf("snapshot=%+v", snap)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fingerprint=%q", snap.SecretFingerprint)
	}
	if strings.Contains(snap.String(), "sk-test") {
		t.Fatal(snap.String())
	}
}

func TestCurrentSchemaModelWriteAndUnsupportedFields(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Model: "custom:gpt-5-0"}); err != nil {
		t.Fatal(err)
	}
	_, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Model: "custom:fixture-next"})
	if err == nil || exitcode.From(err) != exitcode.Usage || !strings.Contains(err.Error(), "customModels") {
		t.Fatalf("unknown custom model: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".factory", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	defaults, ok := raw["sessionDefaultSettings"].(map[string]any)
	if !ok || defaults["model"] != "custom:gpt-5-0" {
		t.Fatalf("model was not written under sessionDefaultSettings: %v", raw)
	}
	if _, ok := raw["model"]; ok {
		t.Fatalf("legacy root model key was written: %v", raw)
	}
	for _, desired := range []model.Desired{{Provider: "openai"}, {SecretRef: "FACTORY_KEY"}} {
		_, err := (Adapter{}).WriteFields(fsx.Local{}, home, desired)
		if err == nil || exitcode.From(err) != exitcode.Usage {
			t.Fatalf("desired=%+v err=%v", desired, err)
		}
	}
}

func TestUnsupportedDesiredFieldsSkipSyncProvider(t *testing.T) {
	got := Adapter{}.UnsupportedDesiredFields()
	want := map[string]bool{"provider": true, "secret-ref": true}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, name := range got {
		if !want[name] {
			t.Fatalf("unexpected field %q in %v", name, got)
		}
	}
}
