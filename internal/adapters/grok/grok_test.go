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

func writeGrokConfig(t *testing.T, home, data string) string {
	t.Helper()
	path := filepath.Join(home, ".grok", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestSecretWritesFailClosedWithoutSelectedModel(t *testing.T) {
	home := t.TempDir()
	path := writeGrokConfig(t, home, `[model."grok-4.5"]
model = "grok-4.5"
base_url = "https://api.x.ai/v1"
`)
	before, _ := os.ReadFile(path)

	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "XAI_KEY"}); err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("secretRef write err=%v", err)
	}
	if err := (Adapter{}).WriteSecret(fsx.Local{}, home, "XAI_KEY", "sk-test-bbb"); err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("bearer write err=%v", err)
	}
	if err := (Adapter{}).ValidateSecretWrite(fsx.Local{}, home, model.Desired{}); err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("validate err=%v", err)
	}
	if err := (Adapter{}).ValidateSecretWrite(fsx.Local{}, home, model.Desired{Model: "grok-4.6"}); err != nil {
		t.Fatalf("pending model should satisfy the selector: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatalf("selector-less write mutated config:\n%s", after)
	}
}

func TestSecretWritesTargetSelectedModel(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	path := filepath.Join(home, ".grok", "config.toml")
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "XAI_KEY"}); err != nil {
		t.Fatal(err)
	}
	if err := (Adapter{}).WriteSecret(fsx.Local{}, home, "", "sk-test-bbb"); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.SecretRef != "XAI_KEY" || snap.SecretFingerprint != secret.Fingerprint("sk-test-bbb") {
		t.Fatalf("snapshot=%+v", snap)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "[model.default]") {
		t.Fatalf("wrote a dead default section:\n%s", data)
	}
}

func TestSecretRefWriteWithPendingModel(t *testing.T) {
	home := t.TempDir()
	writeGrokConfig(t, home, `[model."grok-4.6"]
model = "grok-4.6"
base_url = "https://api.x.ai/v1"
`)
	if _, err := (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{Model: "grok-4.6", SecretRef: "XAI_KEY"}); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "grok-4.6" || snap.SecretRef != "XAI_KEY" {
		t.Fatalf("snapshot=%+v", snap)
	}
}
