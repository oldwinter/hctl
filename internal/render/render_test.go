package render

import (
	"errors"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
)

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestHarnessesTableUsesFingerprintOnly(t *testing.T) {
	s := model.Snapshot{
		Name:              "codex",
		Provider:          "custom",
		DefaultModel:      "gpt-5",
		BaseURLHost:       "api.example.com",
		SecretFingerprint: secret.Fingerprint("sk-test-aaa"),
		SecretPresent:     true,
	}
	var buf strings.Builder
	if err := HarnessesTable(&buf, []model.Snapshot{s}, false); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "sk-test-aaa") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "sha256:"+s.SecretFingerprint) {
		t.Fatal(out)
	}
}

func TestJSONRedactsIfSecretSlipsIn(t *testing.T) {
	var buf strings.Builder
	if err := JSON(&buf, map[string]string{"oops": "sk-test-aaa"}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "sk-test-aaa") {
		t.Fatal(buf.String())
	}
}

func TestContextsTableEmptyNamesSetContext(t *testing.T) {
	var buf strings.Builder
	if err := ContextsTable(&buf, &config.File{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "No contexts configured.") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "hctl config set-context box --kind ssh") {
		t.Fatal(out)
	}
	if strings.Contains(out, "CURRENT") {
		t.Fatal(out)
	}
}

func TestTableWritersSurfaceOutputErrors(t *testing.T) {
	w := failWriter{}
	snaps := []model.Snapshot{{Name: "codex", Installed: true}}
	if err := HarnessesTable(w, snaps, false); err == nil {
		t.Fatal("HarnessesTable")
	}
	if err := HarnessesTable(w, snaps, true); err == nil {
		t.Fatal("HarnessesTable wide")
	}
	if err := ModelsTable(w, snaps); err == nil {
		t.Fatal("ModelsTable")
	}
	if err := DoctorTable(w, []model.DoctorCheck{{Name: "codex"}}); err == nil {
		t.Fatal("DoctorTable")
	}
	if err := ContextsTable(w, config.Default()); err == nil {
		t.Fatal("ContextsTable")
	}
	if err := ContextsTable(w, &config.File{}); err == nil {
		t.Fatal("ContextsTable empty")
	}
	if err := JSON(w, snaps); err == nil {
		t.Fatal("JSON")
	}
	if err := Describe(w, "mba", snaps[0]); err == nil {
		t.Fatal("Describe")
	}
	if err := Diff(w, "a", "b", nil); err == nil {
		t.Fatal("Diff empty")
	}
	if err := Diff(w, "a", "b", []model.FieldDiff{{Field: "model", A: "x", B: "y"}}); err == nil {
		t.Fatal("Diff")
	}
	if err := ApplyReport(w, model.ApplyReport{Changes: []model.Change{{Harness: "codex", Field: "model"}}}); err == nil {
		t.Fatal("ApplyReport")
	}
}
