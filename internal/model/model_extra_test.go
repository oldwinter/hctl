package model

import (
	"strings"
	"testing"
)

func TestSnapshotDesiredDoctor(t *testing.T) {
	s := Snapshot{Name: "x", Installed: true, SecretRef: "E"}
	if !strings.Contains(s.String(), "env:E") {
		t.Fatal(s.String())
	}
	s2 := Snapshot{Name: "y", SecretPresent: true}
	if !strings.Contains(s2.String(), "present") {
		t.Fatal(s2.String())
	}
	if !(Desired{}.Empty()) || (Desired{Model: "m"}.Empty()) {
		t.Fatal("Empty")
	}
	if !(DoctorCheck{Onboarding: "ok", Config: "ok"}.Healthy()) {
		t.Fatal("healthy")
	}
	if (DoctorCheck{Onboarding: "ok", Config: "error"}.Healthy()) {
		t.Fatal("unhealthy")
	}
	_ = DiffSnapshots(Snapshot{DefaultModel: "a"}, Snapshot{DefaultModel: "b"})
}
