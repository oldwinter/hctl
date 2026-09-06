package render

import (
	"strings"
	"testing"

	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

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
	if err := HarnessesTable(&buf, []model.Snapshot{s}); err != nil {
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
