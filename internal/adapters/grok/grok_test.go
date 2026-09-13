package grok

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

func TestWriteModelClonesCurrentTable(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Model: "grok-4.5"}); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "grok-4.5" || snap.BaseURLHost != "api.x.ai" {
		t.Fatalf("snapshot=%+v", snap)
	}
	if snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("fp=%q", snap.SecretFingerprint)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".grok", "config.toml"))
	if !strings.Contains(string(data), `[model."grok-4.6"]`) || !strings.Contains(string(data), `[model."grok-4.5"]`) {
		t.Fatalf("expected cloned table:\n%s", data)
	}
}

func TestWriteModelRejectsOrphanWhenNoTableToCopy(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".grok"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".grok", "config.toml"), []byte("[models]\ndefault = \"gone\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Model: "grok-4.5"})
	if err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("err=%v", err)
	}
}
