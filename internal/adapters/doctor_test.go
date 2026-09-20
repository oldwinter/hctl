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

func TestDoctorOnboardingInspectWithoutParseError(t *testing.T) {
	checks := Doctor([]model.Snapshot{
		{Name: "codex"},
		{Name: "claude", ConfigFound: true, ParseError: "broken settings"},
	})
	byName := map[string]model.DoctorCheck{}
	for _, c := range checks {
		byName[c.Name] = c
	}
	codex := byName["codex"]
	if codex.Onboarding != "needed" {
		t.Fatalf("codex onboarding: %#v", codex)
	}
	if !strings.Contains(codex.Message, "no config file") {
		t.Fatalf("codex reasons: %q", codex.Message)
	}
	if !strings.Contains(codex.Message, "Inspect: hctl describe harness codex") {
		t.Fatalf("codex inspect: %q", codex.Message)
	}
	claude := byName["claude"]
	if claude.Message != "broken settings" {
		t.Fatalf("parse error should stay the message: %#v", claude)
	}
	if strings.Contains(claude.Message, "Inspect:") {
		t.Fatalf("parse error should not add inspect: %q", claude.Message)
	}
}
