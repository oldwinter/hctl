package hermes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
)

func TestSecretRefWritesRequireObservableProvider(t *testing.T) {
	for _, tc := range []struct {
		name, config string
		refused      bool
	}{
		{"auto", "model:\n  provider: auto\nproviders:\n  custom:\n    key_env: ORIGINAL_CUSTOM_KEY\n", true},
		{"multiple-unselected", "model: fixture-model\nproviders:\n  custom:\n    key_env: ORIGINAL_CUSTOM_KEY\n  other:\n    key_env: ORIGINAL_OTHER_KEY\n", true},
		{"empty", "", true},
		{"explicit", "model:\n  provider: custom\nproviders:\n  custom:\n    key_env: ORIGINAL_CUSTOM_KEY\n", false},
		{"custom-prefix", "model:\n  provider: custom:gateway\nproviders:\n  gateway:\n    key_env: ORIGINAL_KEY\n", false},
		{"single-provider", "model: fixture-model\nproviders:\n  gateway:\n    key_env: ORIGINAL_KEY\n", false},
	} {
		for _, fieldWrite := range []bool{false, true} {
			method := "secret-copy"
			if fieldWrite {
				method = "field-write"
			}
			t.Run(tc.name+"/"+method, func(t *testing.T) {
				home := t.TempDir()
				path := writeConfig(t, home, tc.config)
				var err error
				if fieldWrite {
					_, err = (Adapter{}).WriteFields(fsx.Local{}, home, model.Desired{SecretRef: "NEW_KEY"})
				} else {
					err = (Adapter{}).WriteSecret(fsx.Local{}, home, "NEW_KEY", "")
				}
				if tc.refused {
					if exitcode.From(err) != exitcode.Usage {
						t.Errorf("unresolved provider must fail with usage before mutation: %v", err)
					}
					if data, err := os.ReadFile(path); err != nil || string(data) != tc.config {
						t.Errorf("refused write changed inactive configuration: %v", err)
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					snap, err := (Adapter{}).Read(fsx.Local{}, home)
					if err != nil || snap.SecretRef != "NEW_KEY" {
						t.Errorf("written reference is not observable: ref=%q err=%v", snap.SecretRef, err)
					}
				}
				if _, err := os.Stat(filepath.Join(home, ".hermes", ".env")); !os.IsNotExist(err) {
					t.Errorf("ref-only operation created an env file: %v", err)
				}
			})
		}
	}
}

func TestAutoProviderBearerCopyRemainsSupported(t *testing.T) {
	home := t.TempDir()
	config := "model:\n  provider: auto\nproviders:\n  custom:\n    key_env: ORIGINAL_CUSTOM_KEY\n"
	path := writeConfig(t, home, config)
	if err := (Adapter{}).WriteSecret(fsx.Local{}, home, "HERMES_API_KEY", "sk-test-aaa"); err != nil {
		t.Fatal(err)
	}
	snap, err := (Adapter{}).Read(fsx.Local{}, home)
	if err != nil || snap.SecretFingerprint != secret.Fingerprint("sk-test-aaa") {
		t.Fatalf("bearer copy did not remain observable: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != config {
		t.Fatalf("bearer copy changed provider configuration: %v", err)
	}
}
