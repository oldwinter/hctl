package adapters

import (
	"testing"

	"github.com/oldwinter/hctl/internal/model"
)

func TestDoctorDriftAppendsMessage(t *testing.T) {
	checks := Doctor([]model.Snapshot{
		{Name: "a", Installed: true, ConfigFound: true, DefaultModel: "m", Notes: []string{"onboarding needed"}, BaseURLHost: "h.test", SecretFingerprint: "aaa"},
		{Name: "b", Installed: true, ConfigFound: true, DefaultModel: "m", BaseURLHost: "h.test", SecretFingerprint: "bbb"},
		{Name: "c", Installed: true, ConfigFound: true, DefaultModel: "m", Notes: []string{"just a note"}, BaseURLHost: ""},
	})
	if len(checks) < 2 {
		t.Fatal(checks)
	}
	found := false
	for _, c := range checks {
		if c.Drift == "key-drift" && c.Message != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%#v", checks)
	}
}
