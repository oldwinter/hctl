package model

import (
	"fmt"
	"strings"
)

// Snapshot is the unified read model for one harness on one machine/home.
// It never stores plaintext secrets — only a fingerprint and/or env-var ref.
type Snapshot struct {
	Name              string            `json:"name"`
	Installed         bool              `json:"installed"`
	InstalledPath     string            `json:"installedPath,omitempty"`
	Version           string            `json:"version,omitempty"`
	ConfigPaths       []string          `json:"configPaths"`
	ConfigFound       bool              `json:"configFound"`
	Provider          string            `json:"provider,omitempty"`
	BaseURLHost       string            `json:"baseUrlHost,omitempty"`
	DefaultModel      string            `json:"defaultModel,omitempty"`
	Effort            string            `json:"effort,omitempty"`
	Aliases           map[string]string `json:"aliases,omitempty"`
	SecretFingerprint string            `json:"secretFingerprint,omitempty"`
	SecretPresent     bool              `json:"secretPresent"`
	SecretRef         string            `json:"secretRef,omitempty"`
	Notes             []string          `json:"notes,omitempty"`
	ParseError        string            `json:"parseError,omitempty"`
}

// String is a one-line inventory summary. It must never include raw keys.
func (s Snapshot) String() string {
	secret := "-"
	switch {
	case s.SecretFingerprint != "":
		secret = "sha256:" + s.SecretFingerprint
	case s.SecretRef != "":
		secret = "env:" + s.SecretRef
	case s.SecretPresent:
		secret = "present"
	}
	installed := "no"
	if s.Installed {
		installed = "yes"
	}
	return fmt.Sprintf("%s installed=%s provider=%s model=%s host=%s secret=%s",
		s.Name, installed, dash(s.Provider), dash(s.DefaultModel), dash(s.BaseURLHost), secret)
}

func dash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

// DoctorCheck is one row of `harnessctl doctor`.
type DoctorCheck struct {
	Name       string `json:"name"`
	Installed  string `json:"installed"`
	Config     string `json:"config"`
	Key        string `json:"key"`
	Onboarding string `json:"onboarding"`
	Message    string `json:"message,omitempty"`
}

// Healthy reports whether this harness looks ready to use.
func (c DoctorCheck) Healthy() bool {
	return c.Onboarding == "ok" && c.Config != "error"
}

// FieldDiff is one differing field between two snapshots.
type FieldDiff struct {
	Field string `json:"field"`
	A     string `json:"a"`
	B     string `json:"b"`
}

// DiffSnapshots compares inventory fields that matter for v0.1 (never secrets).
func DiffSnapshots(a, b Snapshot) []FieldDiff {
	pairs := []struct {
		field string
		av    string
		bv    string
	}{
		{"defaultModel", a.DefaultModel, b.DefaultModel},
		{"provider", a.Provider, b.Provider},
		{"baseUrlHost", a.BaseURLHost, b.BaseURLHost},
		{"effort", a.Effort, b.Effort},
		{"secretFingerprint", a.SecretFingerprint, b.SecretFingerprint},
		{"secretRef", a.SecretRef, b.SecretRef},
		{"secretPresent", fmt.Sprintf("%t", a.SecretPresent), fmt.Sprintf("%t", b.SecretPresent)},
		{"configFound", fmt.Sprintf("%t", a.ConfigFound), fmt.Sprintf("%t", b.ConfigFound)},
		{"version", a.Version, b.Version},
	}
	var out []FieldDiff
	for _, p := range pairs {
		if p.av != p.bv {
			out = append(out, FieldDiff{Field: p.field, A: p.av, B: p.bv})
		}
	}
	return out
}
