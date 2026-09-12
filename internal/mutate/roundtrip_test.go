package mutate

import (
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/testutil"
)

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
