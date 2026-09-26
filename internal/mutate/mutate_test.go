package mutate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/adapters/claude"
	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/adapters/grok"
	"github.com/oldwinter/hctl/internal/adapters/hermes"
	"github.com/oldwinter/hctl/internal/adapters/pi"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestApplyMustUpdateEndpointPathOnSameHost(t *testing.T) {
	testApplyEndpointChange(t, "https://gateway.example/old/v1", "https://gateway.example/new/v1")
}

func TestApplyMustUpdateEndpointSchemeOnSameHost(t *testing.T) {
	testApplyEndpointChange(t, "http://gateway.example/v1", "https://gateway.example/v1")
}

func testApplyEndpointChange(t *testing.T, before, want string) {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".hermes"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".hermes", "config.yaml")
	original := "model:\n  provider: custom\nproviders:\n  custom:\n    base_url: " + before + "\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	req := Request{
		Adapter: hermes.Adapter{}, FS: fsx.Local{}, Home: home,
		BackupDir: t.TempDir(), Desired: model.Desired{BaseURL: want}, DryRun: true,
	}
	rep, err := Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Changes) != 1 || len(rep.Backups) != 0 || rep.Verified {
		t.Fatalf("dry-run report = %#v", rep)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != original {
		t.Fatalf("dry-run changed config: %v", err)
	}
	entries, err := os.ReadDir(req.BackupDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("dry-run created backups: %v", err)
	}
	req.DryRun = false
	rep, err = Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	got, err := (hermes.Adapter{}).ReadEndpoint(fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if got != want || len(rep.Changes) != 1 || !rep.Verified || len(rep.Backups) != 1 {
		t.Fatalf("endpoint=%q want=%q report=%#v", got, want, rep)
	}
	backup, err := os.ReadFile(rep.Backups[0])
	if err != nil || string(backup) != original {
		t.Fatalf("backup does not preserve original config: %v", err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rep, err = Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Changes) != 0 || len(rep.Backups) != 0 || rep.Verified {
		t.Fatalf("unchanged full endpoint should no-op: %#v", rep)
	}
	data, err = os.ReadFile(path)
	if err != nil || string(data) != string(written) {
		t.Fatalf("idempotent apply changed config: %v", err)
	}
	entries, err = os.ReadDir(req.BackupDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("idempotent apply created extra backups: %v", err)
	}
}

// This writer changes the model but silently ignores the requested endpoint.
type ignoringEndpointAdapter struct {
	hermes.Adapter
}

func (a ignoringEndpointAdapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	d.BaseURL = ""
	return a.Adapter.WriteFields(fsys, home, d)
}

func TestApplyVerifiesFullEndpoint(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	ad := hermes.Adapter{}
	before := "https://user:sk-test-aaa@gateway.example/old/v1?token=sk-test-aaa"
	want := "https://user:sk-test-bbb@gateway.example/new/v1?token=sk-test-bbb"
	if _, err := ad.WriteFields(fsx.Local{}, home, model.Desired{BaseURL: before}); err != nil {
		t.Fatal(err)
	}
	rep, err := Apply(Request{
		Adapter: ignoringEndpointAdapter{}, FS: fsx.Local{}, Home: home, BackupDir: t.TempDir(),
		Desired: model.Desired{Model: "new-model", BaseURL: want},
	})
	if err == nil || exitcode.From(err) != exitcode.Verify || !strings.Contains(err.Error(), "baseUrl verify failed") || rep.Verified {
		t.Fatalf("expected endpoint verification failure, got %v (verified=%v)", err, rep.Verified)
	}
	if strings.Contains(err.Error(), "sk-test") || strings.Contains(err.Error(), "user:") || strings.Contains(err.Error(), "/v1") {
		t.Fatal("verification error exposed endpoint credentials or path")
	}
}

func TestApplySetModelRoundTrip(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	bak := t.TempDir()
	t.Setenv("HARNESSCTL_BACKUP_DIR", bak)
	ad := codex.Adapter{}
	rep, err := Apply(Request{
		Adapter:   ad,
		FS:        fsx.Local{},
		Home:      home,
		BackupDir: bak,
		Desired:   model.Desired{Model: "o4-mini"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Verified {
		t.Fatal("expected verify")
	}
	snap, err := adapters.ReadOne(ad, fsx.Local{}, home)
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "o4-mini" {
		t.Fatalf("got %q", snap.DefaultModel)
	}
	if snap.Provider != "custom" {
		t.Fatalf("provider churned: %q", snap.Provider)
	}
	data, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if !strings.Contains(string(data), "experimental_bearer_token") {
		t.Fatal("lost unrelated secret field")
	}
	if !strings.Contains(string(data), "sk-test-aaa") {
		t.Fatal("lost bearer token (must stay in file, never printed)")
	}
	ents, _ := os.ReadDir(bak)
	if len(ents) == 0 {
		t.Fatal("expected backup")
	}
}

func TestApplyIdempotentSkipsBackupAndWrite(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	bak := t.TempDir()
	ad := codex.Adapter{}
	req := Request{
		Adapter:   ad,
		FS:        fsx.Local{},
		Home:      home,
		BackupDir: bak,
		Desired:   model.Desired{Model: "o4-mini"},
	}
	if _, err := Apply(req); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(bak)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) == 0 {
		t.Fatal("expected first apply to backup")
	}
	before, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Apply(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Changes) != 0 || len(rep.Backups) != 0 || rep.Verified {
		t.Fatalf("converged apply should no-op: %#v", rep)
	}
	after, err := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("idempotent apply rewrote config")
	}
	ents2, err := os.ReadDir(bak)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents2) != len(ents) {
		t.Fatalf("idempotent apply created extra backups: %d -> %d", len(ents), len(ents2))
	}
}

func TestDryRunDoesNotWrite(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	before, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	rep, err := Apply(Request{
		Adapter: codex.Adapter{},
		FS:      fsx.Local{},
		Home:    home,
		Desired: model.Desired{Model: "nope"},
		DryRun:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(home, ".codex", "config.toml"))
	if string(before) != string(after) {
		t.Fatal("dry-run mutated file")
	}
	if len(rep.Changes) != 1 {
		t.Fatalf("changes = %#v", rep.Changes)
	}
	wantPath := filepath.Join(home, ".codex", "config.toml")
	if rep.Changes[0].Path != wantPath {
		t.Fatalf("dry-run Path = %q want %q", rep.Changes[0].Path, wantPath)
	}
}

func TestSetProviderUnsupportedClaudeAndGrok(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	cases := []struct {
		ad   adapters.Adapter
		name string
	}{
		{claude.Adapter{}, "claude"},
		{grok.Adapter{}, "grok"},
	}
	for _, tc := range cases {
		beforeClaude, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
		beforeGrok, _ := os.ReadFile(filepath.Join(home, ".grok", "config.toml"))
		_, err := Apply(Request{
			Adapter: tc.ad,
			FS:      fsx.Local{},
			Home:    home,
			Desired: model.Desired{Provider: "custom"},
			DryRun:  true,
		})
		if err == nil {
			t.Fatalf("%s: expected unsupported error", tc.name)
		}
		if exitcode.From(err) != exitcode.Usage {
			t.Fatalf("%s: code=%d err=%v", tc.name, exitcode.From(err), err)
		}
		if !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("%s: err=%v", tc.name, err)
		}
		afterClaude, _ := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
		afterGrok, _ := os.ReadFile(filepath.Join(home, ".grok", "config.toml"))
		if string(beforeClaude) != string(afterClaude) || string(beforeGrok) != string(afterGrok) {
			t.Fatalf("%s: dry-run/error mutated files", tc.name)
		}
	}
}

func TestApplyClaudeBacksUpOnlyWrittenSettings(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	bak := t.TempDir()
	rep, err := Apply(Request{
		Adapter:   claude.Adapter{},
		FS:        fsx.Local{},
		Home:      home,
		BackupDir: bak,
		Desired:   model.Desired{Model: "claude-opus-4"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Backups) != 1 || !strings.Contains(rep.Backups[0], "settings.json") {
		t.Fatalf("backups=%v", rep.Backups)
	}
	for _, p := range rep.Backups {
		if strings.Contains(p, ".claude.json") {
			t.Fatalf("backed up unchanged onboarding file: %v", rep.Backups)
		}
	}
}

func TestPreflightRejectsParseErrorBeforeBackupOrWrite(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	backupDir := t.TempDir()
	settingsPath := filepath.Join(home, ".pi", "agent", "settings.json")
	authPath := filepath.Join(home, ".pi", "agent", "auth.json")
	before, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte(`{"sub2api":`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = Apply(Request{
		Adapter:   pi.Adapter{},
		FS:        fsx.Local{},
		Home:      home,
		BackupDir: backupDir,
		Desired:   model.Desired{Model: "must-not-be-written"},
	})
	if err == nil || exitcode.From(err) != exitcode.Parse || strings.Contains(err.Error(), `{"sub2api":`) {
		t.Fatalf("expected sanitized parse error, got %v", err)
	}
	after, readErr := os.ReadFile(settingsPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(before) {
		t.Fatal("settings changed after parse-error preflight")
	}
	entries, readErr := os.ReadDir(backupDir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("preflight created backups: %v", entries)
	}
}
