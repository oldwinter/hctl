package render

import (
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/model"
)

func TestAllRenderHelpers(t *testing.T) {
	snaps := []model.Snapshot{{
		Name: "codex", Installed: true, InstalledPath: "/bin/codex", Version: "1",
		Provider: "p", DefaultModel: "m", Effort: "high", BaseURLHost: "h",
		SecretPresent: true, ConfigPaths: []string{"/a"}, Aliases: map[string]string{"k": "v"},
		ParseError: "pe", Notes: []string{"n1"},
	}, {
		Name: "empty", SecretRef: "ENV",
	}, {
		Name: "fp", SecretFingerprint: "abc",
	}}
	var buf strings.Builder
	if err := HarnessesTable(&buf, snaps, true); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := ModelsTable(&buf, snaps); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := Describe(&buf, "mba", snaps[0]); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "/bin/codex (1)") {
		t.Fatal(buf.String())
	}
	buf.Reset()
	if err := Describe(&buf, "mba", model.Snapshot{Name: "x", Installed: true, InstalledPath: "/x"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Installed:         /x\n") {
		t.Fatal(buf.String())
	}
	buf.Reset()
	if err := DoctorTable(&buf, []model.DoctorCheck{{Name: "c", Installed: "yes", Config: "ok", Key: "ok", Onboarding: "ok", Message: "m"}}); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	f := &config.File{CurrentContext: "mba", Contexts: []config.NamedContext{
		{Name: "mba", Context: config.Context{Kind: "local", Home: "/h"}},
		{Name: "box", Context: config.Context{Kind: "ssh", SSH: "u@h"}},
	}}
	if err := ContextsTable(&buf, f); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := Diff(&buf, "a", "b", nil); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "no differences\n" {
		t.Fatal(buf.String())
	}
	buf.Reset()
	if err := Diff(&buf, "a", "b", []model.FieldDiff{{Field: "model", A: "", B: "x"}}); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := ApplyReport(&buf, model.ApplyReport{DryRun: true}); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := ApplyReport(&buf, model.ApplyReport{Verified: true, Changes: []model.Change{{Harness: "h", Field: "model", From: "", To: "m", Path: "p"}}, Backups: []string{"bak"}}); err != nil {
		t.Fatal(err)
	}
	if secretCell(model.Snapshot{}) != "-" {
		t.Fatal(secretCell(model.Snapshot{}))
	}
	if installedLine(model.Snapshot{}) != "no" {
		t.Fatal(installedLine(model.Snapshot{}))
	}
	if yesNo(false) != "no" || yesNo(true) != "yes" {
		t.Fatal("yesNo")
	}
	if empty("") != "-" || empty("x") != "x" {
		t.Fatal("empty")
	}
	if dash("  ") != "-" {
		t.Fatal(dash("  "))
	}
}
