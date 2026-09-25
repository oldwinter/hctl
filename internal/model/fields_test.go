package model

import "testing"

func TestChangesFromDesired(t *testing.T) {
	snap := Snapshot{Name: "codex", DefaultModel: "old", Provider: "custom", SecretRef: "OLD_KEY", ConfigPaths: []string{"/tmp/codex.toml"}}
	d := Desired{Model: "new", Provider: "custom", SecretRef: "NEW_KEY"}
	ch := ChangesFromDesired("codex", snap, d)
	if len(ch) != 2 {
		t.Fatalf("changes = %#v", ch)
	}
	if ch[0].Field != "model" || ch[0].From != "old" || ch[0].To != "new" || ch[0].Path != "/tmp/codex.toml" {
		t.Fatalf("model = %#v", ch[0])
	}
	if ch[1].Field != "secretRef" || ch[1].To != "NEW_KEY" {
		t.Fatalf("secretRef = %#v", ch[1])
	}
}

func TestDesiredEmptyUsesFieldTable(t *testing.T) {
	if !(Desired{}).Empty() {
		t.Fatal("zero desired should be empty")
	}
	if (Desired{Model: "x"}).Empty() {
		t.Fatal("model should not be empty")
	}
}

func TestProjectAndKnownSyncField(t *testing.T) {
	snap := Snapshot{DefaultModel: "m", Provider: "p", SecretRef: "R"}
	got := Project(snap, "model", "secret-ref")
	if got.Model != "m" || got.Provider != "" || got.SecretRef != "R" {
		t.Fatalf("project = %#v", got)
	}
	if !KnownSyncField("secret") || !KnownSyncField("secretRef") || KnownSyncField("effort") {
		t.Fatal("sync field vocabulary")
	}
	if !KnownSyncField("base-url") || !KnownSyncField("baseUrl") {
		t.Fatal("base-url should be a sync field")
	}
}

func TestBaseURLChangesCompareAtHostLevel(t *testing.T) {
	snap := Snapshot{Name: "hermes", BaseURLHost: "gateway.example"}
	if ch := ChangesFromDesired("hermes", snap, Desired{BaseURL: "https://gateway.example/team/v1"}); len(ch) != 0 {
		t.Fatalf("same-host endpoint should produce no change: %#v", ch)
	}
	ch := ChangesFromDesired("hermes", snap, Desired{BaseURL: "https://other.example/v1"})
	if len(ch) != 1 || ch[0].Field != "baseUrl" || ch[0].From != "gateway.example" || ch[0].To != "https://other.example/v1" {
		t.Fatalf("baseUrl change = %#v", ch)
	}
	if (Desired{BaseURL: "https://x"}).Empty() {
		t.Fatal("baseUrl should count as non-empty")
	}
	if got := Project(snap, "base-url"); got.BaseURL != "" {
		t.Fatalf("projection must not copy the host into baseUrl: %#v", got)
	}
}
