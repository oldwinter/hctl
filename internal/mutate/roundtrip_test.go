package mutate

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestCodexTOMLTableBoundariesRoundTrip(t *testing.T) {
	const src = `model_provider = "custom"

[[profiles]] # preserve profile
model = "profile-model" # profile default

[model_providers.custom] # preserve provider
env_key = "OLD_KEY" # provider env
base_url = "https://api.example.com/v1"
`
	cases := []struct {
		name    string
		desired model.Desired
	}{
		{"root-model", model.Desired{Model: "new-model"}},
		{"commented-provider", model.Desired{SecretRef: "NEW_KEY"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			configDir := filepath.Join(home, ".codex")
			if err := os.MkdirAll(configDir, 0o700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(configDir, "config.toml")
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			var want map[string]any
			if err := toml.Unmarshal([]byte(src), &want); err != nil {
				t.Fatal(err)
			}
			if tc.desired.Model != "" {
				want["model"] = tc.desired.Model
			}
			if tc.desired.SecretRef != "" {
				providers := want["model_providers"].(map[string]any)
				providers["custom"].(map[string]any)["env_key"] = tc.desired.SecretRef
			}
			rep, err := Apply(Request{
				Adapter:   codex.Adapter{},
				FS:        fsx.Local{},
				Home:      home,
				BackupDir: t.TempDir(),
				Desired:   tc.desired,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !rep.Verified || len(rep.Backups) != 1 {
				t.Fatalf("expected verified write with one backup: %#v", rep)
			}
			backup, err := os.ReadFile(rep.Backups[0])
			if err != nil {
				t.Fatal(err)
			}
			if string(backup) != src {
				t.Fatal("backup differs from original config")
			}
			out, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := toml.Unmarshal(out, &got); err != nil {
				t.Fatalf("invalid TOML after apply: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
			for _, comment := range []string{"# preserve profile", "# profile default", "# preserve provider", "# provider env"} {
				if !strings.Contains(string(out), comment) {
					t.Errorf("lost comment %q", comment)
				}
			}
		})
	}
}

func TestAllWritersSetModel(t *testing.T) {
	home := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	bak := t.TempDir()
	want := map[string]string{
		"codex":    "round-trip-codex",
		"claude":   "round-trip-claude",
		"grok":     "round-trip-grok",
		"hermes":   "round-trip-hermes",
		"opencode": "acme/round-trip",
	}
	for name, modelName := range want {
		ad, err := adapters.ByName(name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = Apply(Request{
			Adapter:   ad,
			FS:        fsx.Local{},
			Home:      home,
			BackupDir: bak,
			Desired:   model.Desired{Model: modelName},
		})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		snap, err := adapters.ReadOne(ad, fsx.Local{}, home)
		if err != nil {
			t.Fatal(err)
		}
		if snap.DefaultModel != modelName {
			t.Fatalf("%s model = %q", name, snap.DefaultModel)
		}
		if strings.Contains(snap.String(), "sk-test") {
			t.Fatalf("%s leaked", name)
		}
	}
}
