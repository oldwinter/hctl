package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

func TestMetaAndWritePeekSecret(t *testing.T) {
	a := Adapter{}
	if a.Aliases()[0] != "claude-code" {
		t.Fatal(a.Aliases())
	}
	if a.BinaryNames()[0] != "claude" {
		t.Fatal(a.BinaryNames())
	}
	if len(a.ConfigRelPaths()) != 2 {
		t.Fatal(a.ConfigRelPaths())
	}
	home := t.TempDir()
	_, err := a.WriteFields(fsx.Local{}, home, model.Desired{Provider: "x"})
	if err == nil || exitcode.From(err) != exitcode.Usage {
		t.Fatalf("%v", err)
	}
	paths, err := a.WriteFields(fsx.Local{}, home, model.Desired{Model: "claude-opus", SecretRef: "$ANTHROPIC_API_KEY"})
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 {
		t.Fatal(paths)
	}
	ref, val, err := a.PeekSecret(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if ref != "ANTHROPIC_API_KEY" || val != "" {
		t.Fatalf("%q %q", ref, val)
	}
	if err := a.WriteSecret(fsx.Local{}, home, "", "sk-test-claude"); err != nil {
		t.Fatal(err)
	}
	ref, val, err = a.PeekSecret(fsx.Local{}, home)
	if err != nil || val != "sk-test-claude" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	_ = ref
	if err := a.WriteSecret(fsx.Local{}, home, "$OTHER", ""); err != nil {
		t.Fatal(err)
	}
	// empty peek
	empty := t.TempDir()
	ref, val, err = a.PeekSecret(fsx.Local{}, empty)
	if err != nil || ref != "" || val != "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	// bad json peek
	bad := t.TempDir()
	os.MkdirAll(filepath.Join(bad, ".claude"), 0o755)
	os.WriteFile(filepath.Join(bad, ".claude", "settings.json"), []byte("{"), 0o600)
	if _, _, err := a.PeekSecret(fsx.Local{}, bad); err == nil {
		t.Fatal("expected parse err")
	}
	// env nil
	home2 := t.TempDir()
	os.MkdirAll(filepath.Join(home2, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home2, ".claude", "settings.json"), []byte(`{"model":"m"}`), 0o600)
	ref, val, err = a.PeekSecret(fsx.Local{}, home2)
	if err != nil || ref != "" || val != "" {
		t.Fatalf("%q %q %v", ref, val, err)
	}
	// ReadFS parse error + onboarding branches
	home3 := t.TempDir()
	os.MkdirAll(filepath.Join(home3, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home3, ".claude", "settings.json"), []byte(`{bad`), 0o600)
	os.WriteFile(filepath.Join(home3, ".claude.json"), []byte(`{bad`), 0o600)
	snap, err := a.ReadFS(fsx.Local{}, home3)
	if err != nil || snap.ParseError == "" {
		t.Fatalf("%#v %v", snap, err)
	}
	home4 := t.TempDir()
	os.MkdirAll(filepath.Join(home4, ".claude"), 0o755)
	body, _ := json.Marshal(map[string]any{"theme": "dark", "model": "", "env": map[string]string{}})
	os.WriteFile(filepath.Join(home4, ".claude", "settings.json"), body, 0o600)
	os.WriteFile(filepath.Join(home4, ".claude.json"), []byte(`{"theme":"","hasCompletedOnboarding":false}`), 0o600)
	snap, _ = a.ReadFS(fsx.Local{}, home4)
	if len(snap.Notes) == 0 {
		t.Fatal(snap.Notes)
	}
	// env ref + string onboarding true
	home5 := t.TempDir()
	os.MkdirAll(filepath.Join(home5, ".claude"), 0o755)
	os.WriteFile(filepath.Join(home5, ".claude", "settings.json"), []byte(`{"env":{"ANTHROPIC_MODEL":"m","ANTHROPIC_API_KEY":"$KEY"}}`), 0o600)
	os.WriteFile(filepath.Join(home5, ".claude.json"), []byte(`{"theme":"x","hasCompletedOnboardingForNewUsers":"true"}`), 0o600)
	snap, _ = a.ReadFS(fsx.Local{}, home5)
	if snap.SecretRef != "KEY" || snap.DefaultModel != "m" {
		t.Fatalf("%#v", snap)
	}
}
