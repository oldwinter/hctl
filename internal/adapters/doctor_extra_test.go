package adapters

import (
	"testing"

	"github.com/oldwinter/hctl/internal/model"
)

func TestDoctorBranches(t *testing.T) {
	checks := Doctor([]model.Snapshot{
		{Name: "a", Installed: true, ConfigFound: true, DefaultModel: "m", SecretFingerprint: "x", BaseURLHost: "h", Notes: []string{"ok note"}},
		{Name: "b", Installed: false, ConfigFound: false, BaseURLHost: "h", SecretFingerprint: "y"},
		{Name: "c", ConfigFound: true, ParseError: "bad", Notes: []string{"onboarding needed"}},
		{Name: "d", ConfigFound: true, SecretRef: "ENV", Notes: []string{"login not detected"}},
	})
	if len(checks) != 4 {
		t.Fatal(checks)
	}
}
