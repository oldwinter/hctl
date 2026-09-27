package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/hermes"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
)

const refOnlyHermesSource = "model:\n  default: fixture-model\n  provider: custom\nproviders:\n  custom:\n    key_env: MY_HERMES_KEY\n"

func TestSyncHermesRefOnlyRejectsUnresolvedProviderBeforeMutation(t *testing.T) {
	for _, tc := range []struct{ name, config string }{
		{"auto", "model:\n  provider: auto\nproviders:\n  custom:\n    key_env: ORIGINAL_CUSTOM_KEY\n"},
		{"multiple-unselected", "model: fixture-model\nproviders:\n  custom:\n    key_env: ORIGINAL_CUSTOM_KEY\n  other:\n    key_env: ORIGINAL_OTHER_KEY\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, dry := range []bool{true, false} {
				src, dst := t.TempDir(), t.TempDir()
				writeTestFile(t, filepath.Join(src, ".hermes", "config.yaml"), refOnlyHermesSource)
				path := filepath.Join(dst, ".hermes", "config.yaml")
				writeTestFile(t, path, tc.config)
				backups := t.TempDir()
				t.Setenv("HARNESSCTL_BACKUP_DIR", backups)
				args := []string{"--no-probe", "--config", writeLocalContexts(t, src, dst), "sync", "--from", "source", "--to", "destination", "--harness", "hermes", "--fields", "secret"}
				if dry {
					args = append(args, "--dry-run")
				}
				out, err := run(t, args...)
				if exitcode.From(err) != exitcode.Usage {
					t.Errorf("dry=%v: want preflight usage error, got %v", dry, err)
				}
				if strings.Contains(out, "sk-") {
					t.Error("credential appeared in output")
				}
				if data, err := os.ReadFile(path); err != nil || string(data) != tc.config {
					t.Errorf("dry=%v: inactive provider was modified: %v", dry, err)
				}
				if _, err := os.Stat(filepath.Join(dst, ".hermes", ".env")); !os.IsNotExist(err) {
					t.Errorf("env file created on refusal: %v", err)
				}
				assertDirectoryEmpty(t, backups)
			}
		})
	}
}

func TestSyncHermesRefOnlyUsesPendingProvider(t *testing.T) {
	for _, missingConfig := range []bool{false, true} {
		src, dst := t.TempDir(), t.TempDir()
		writeTestFile(t, filepath.Join(src, ".hermes", "config.yaml"), refOnlyHermesSource)
		if !missingConfig {
			writeTestFile(t, filepath.Join(dst, ".hermes", "config.yaml"), "model:\n  provider: auto\nproviders:\n  custom:\n    key_env: ORIGINAL_CUSTOM_KEY\n")
		}
		t.Setenv("HARNESSCTL_BACKUP_DIR", t.TempDir())
		out, err := run(t, "--no-probe", "--config", writeLocalContexts(t, src, dst), "sync", "--from", "source", "--to", "destination", "--harness", "hermes", "--fields", "provider,secret")
		if err != nil || !strings.Contains(out, "verified: ok") {
			t.Fatalf("pending provider transfer: %v", err)
		}
		snap, err := (hermes.Adapter{}).Read(fsx.Local{}, dst)
		if err != nil || snap.Provider != "custom" || snap.SecretRef != "MY_HERMES_KEY" {
			t.Fatalf("pending provider/reference not observable: %+v (%v)", snap, err)
		}
	}
}
