package model

import (
	"strings"
	"testing"
)

func TestStringNeverIncludesRawKey(t *testing.T) {
	s := Snapshot{
		Name:              "codex",
		Provider:          "custom",
		DefaultModel:      "gpt-5",
		SecretFingerprint: "deadbeef",
		SecretPresent:     true,
	}
	out := s.String()
	if strings.Contains(out, "sk-") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "sha256:deadbeef") {
		t.Fatal(out)
	}
}

func TestDiffSnapshots(t *testing.T) {
	a := Snapshot{DefaultModel: "x", Provider: "p", SecretFingerprint: "aaa"}
	b := Snapshot{DefaultModel: "y", Provider: "p", SecretFingerprint: "bbb"}
	d := DiffSnapshots(a, b)
	fields := map[string]bool{}
	for _, x := range d {
		fields[x.Field] = true
	}
	if !fields["defaultModel"] || !fields["secretFingerprint"] {
		t.Fatalf("%#v", d)
	}
	if fields["provider"] {
		t.Fatal("provider should match")
	}
}
