package adapters

import (
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
)

func TestDoctorGroupsKeyDriftByHost(t *testing.T) {
	host := "sub2api.example"
	snaps := []model.Snapshot{
		{Name: "codex", ConfigFound: true, DefaultModel: "gpt-5", BaseURLHost: host, SecretFingerprint: secret.Fingerprint("sk-test-aaa"), SecretPresent: true},
		{Name: "claude", ConfigFound: true, DefaultModel: "claude-sonnet-4", BaseURLHost: host, SecretFingerprint: secret.Fingerprint("sk-test-bbb"), SecretPresent: true},
		{Name: "grok", ConfigFound: true, DefaultModel: "grok-4", BaseURLHost: "api.x.ai", SecretFingerprint: secret.Fingerprint("sk-test-aaa"), SecretPresent: true},
	}
	checks := Doctor(snaps)
	var drift []model.DoctorCheck
	for _, c := range checks {
		if c.Drift == "key-drift" {
			drift = append(drift, c)
		}
	}
	if len(drift) != 2 {
		t.Fatalf("want 2 drift rows, got %#v", checks)
	}
	for _, c := range drift {
		if c.Name != "codex" && c.Name != "claude" {
			t.Fatalf("unexpected drift on %s", c.Name)
		}
		if !strings.Contains(c.Message, "key drift on host sub2api.example (claude,codex)") {
			t.Fatalf("grouped message: %q", c.Message)
		}
	}
}
