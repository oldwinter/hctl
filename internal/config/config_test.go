package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsDefaultMBA(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if f.CurrentContext != "mba" {
		t.Fatalf("current = %q", f.CurrentContext)
	}
	if _, err := f.Get("mba"); err != nil {
		t.Fatal(err)
	}
}

func TestUseContextAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	f := Default()
	f.Contexts = append(f.Contexts, NamedContext{
		Name:    "box",
		Context: Context{Kind: KindSSH, SSH: "user@box", Home: "/home/user"},
	})
	if err := f.UseContext("box"); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm = %o", st.Mode().Perm())
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentContext != "box" {
		t.Fatalf("current = %q", got.CurrentContext)
	}
}

func TestUnknownContext(t *testing.T) {
	if err := Default().UseContext("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestGetenvPrefersHCTL(t *testing.T) {
	t.Setenv("HCTL_CONFIG", "/tmp/hctl.yaml")
	t.Setenv("HARNESSCTL_CONFIG", "/tmp/legacy.yaml")
	if got := Getenv("CONFIG"); got != "/tmp/hctl.yaml" {
		t.Fatalf("Getenv = %q", got)
	}
	t.Setenv("HCTL_CONFIG", "")
	if got := Getenv("CONFIG"); got != "/tmp/legacy.yaml" {
		t.Fatalf("Getenv fallback = %q", got)
	}
}

func TestResolvePathsSources(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HCTL_CONFIG", "")
	t.Setenv("HARNESSCTL_CONFIG", "")

	flag := ResolvePaths("/tmp/from-flag.yaml")
	if flag.Source != SourceFlag || flag.Config != "/tmp/from-flag.yaml" {
		t.Fatalf("flag = %#v", flag)
	}

	t.Setenv("HCTL_CONFIG", "/tmp/from-env.yaml")
	env := ResolvePaths("")
	if env.Source != SourceEnv || env.Config != "/tmp/from-env.yaml" {
		t.Fatalf("env = %#v", env)
	}
	t.Setenv("HCTL_CONFIG", "")
	t.Setenv("HARNESSCTL_CONFIG", "/tmp/from-legacy-env.yaml")
	legacyEnv := ResolvePaths("")
	if legacyEnv.Source != SourceEnv || legacyEnv.Config != "/tmp/from-legacy-env.yaml" {
		t.Fatalf("legacy env = %#v", legacyEnv)
	}
	t.Setenv("HARNESSCTL_CONFIG", "")

	canonical := filepath.Join(home, ".hctl", "config.yaml")
	legacy := filepath.Join(home, ".harnessctl", "config.yaml")
	fresh := ResolvePaths("")
	if fresh.Source != SourceCanonical || fresh.Config != canonical {
		t.Fatalf("fresh = %#v", fresh)
	}

	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}
	gotLegacy := ResolvePaths("")
	if gotLegacy.Source != SourceLegacy || gotLegacy.Config != legacy {
		t.Fatalf("legacy file = %#v", gotLegacy)
	}

	if err := os.MkdirAll(filepath.Dir(canonical), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonical, []byte("canonical"), 0o600); err != nil {
		t.Fatal(err)
	}
	gotCanon := ResolvePaths("")
	if gotCanon.Source != SourceCanonical || gotCanon.Config != canonical {
		t.Fatalf("canonical wins = %#v", gotCanon)
	}
	if _, err := os.Stat(legacy); err != nil {
		t.Fatal("legacy file must stay in place")
	}
	if DefaultPath() != canonical {
		t.Fatalf("DefaultPath = %q", DefaultPath())
	}
}

func TestSaveNoopWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("HCTL_BACKUP_DIR", filepath.Join(dir, "bak"))
	if err := Save(path, Default()); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, Default()); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(filepath.Join(dir, "bak"))
	if err == nil && len(ents) != 0 {
		t.Fatalf("identical save created backups: %v", ents)
	}
	f := Default()
	if err := f.UseContext("mba"); err != nil {
		t.Fatal(err)
	}
	f.Contexts = append(f.Contexts, NamedContext{Name: "box", Context: Context{Kind: KindLocal}})
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestBackupDirEnvAndBesideConfig(t *testing.T) {
	t.Setenv("HCTL_BACKUP_DIR", "/tmp/hctl-bak")
	t.Setenv("HARNESSCTL_BACKUP_DIR", "/tmp/legacy-bak")
	if BackupDir("/x/config.yaml") != "/tmp/hctl-bak" {
		t.Fatalf("hctl env = %q", BackupDir("/x/config.yaml"))
	}
	t.Setenv("HCTL_BACKUP_DIR", "")
	if BackupDir("/x/config.yaml") != "/tmp/legacy-bak" {
		t.Fatalf("legacy env = %q", BackupDir("/x/config.yaml"))
	}
	t.Setenv("HARNESSCTL_BACKUP_DIR", "")
	if BackupDir("/x/config.yaml") != "/x/backups" {
		t.Fatalf("beside = %q", BackupDir("/x/config.yaml"))
	}
}
