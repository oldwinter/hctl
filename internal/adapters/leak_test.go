package adapters

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/oldwinter/harnessctl/internal/render"
	"github.com/oldwinter/harnessctl/internal/secret"
	"github.com/oldwinter/harnessctl/internal/testutil"
)

func TestScanNeverLeaksFixtures(t *testing.T) {
	for _, home := range []string{"home-a", "home-b"} {
		snaps, err := Scan(testutil.Testdata(t, home))
		if err != nil {
			t.Fatal(err)
		}
		var buf strings.Builder
		if err := render.HarnessesTable(&buf, snaps); err != nil {
			t.Fatal(err)
		}
		if err := render.ModelsTable(&buf, snaps); err != nil {
			t.Fatal(err)
		}
		for _, s := range snaps {
			if err := render.Describe(&buf, home, s); err != nil {
				t.Fatal(err)
			}
			buf.WriteString(s.String())
			raw, err := json.Marshal(s)
			if err != nil {
				t.Fatal(err)
			}
			buf.Write(raw)
		}
		out := buf.String()
		for _, leak := range []string{"sk-test-aaa", "sk-test-bbb", "sk-test"} {
			if strings.Contains(out, leak) {
				t.Fatalf("%s output leaked %q:\n%s", home, leak, out)
			}
		}
		if secret.LooksLikeSecret(out) {
			t.Fatalf("%s table/json looks like it still contains a secret:\n%s", home, out)
		}
	}
}

func TestDoctorHomeAReady(t *testing.T) {
	snaps, err := Scan(testutil.Testdata(t, "home-a"))
	if err != nil {
		t.Fatal(err)
	}
	checks := Doctor(snaps)
	ready := map[string]bool{}
	for _, c := range checks {
		ready[c.Name] = c.Config == "ok" && c.Key == "ok" && c.Onboarding == "ok"
	}
	for _, name := range []string{"codex", "claude", "grok", "hermes", "opencode"} {
		if !ready[name] {
			t.Fatalf("%s not ready: %#v", name, checks)
		}
	}
}

func TestDoctorHomeEmptyNeedsOnboarding(t *testing.T) {
	snaps, err := Scan(testutil.Testdata(t, "home-empty"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range Doctor(snaps) {
		if c.Name == "pi" || c.Name == "droid" || c.Name == "cursor-agent" {
			continue
		}
		if c.Onboarding != "needed" {
			t.Fatalf("%s onboarding = %q", c.Name, c.Onboarding)
		}
	}
}
