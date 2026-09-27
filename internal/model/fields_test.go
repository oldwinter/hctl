package model

import "testing"

func TestChangesFromDesired(t *testing.T) {
	snap := Snapshot{Name: "codex", DefaultModel: "old", Provider: "custom", SecretRef: "OLD_KEY", ConfigPaths: []string{"/tmp/codex.toml"}}
	d := Desired{Model: "new", Provider: "custom", SecretRef: "NEW_KEY"}
	ch := ChangesFromDesired("codex", snap, d, nil)
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
	if ch := ChangesFromDesired("hermes", snap, Desired{BaseURL: "https://gateway.example/team/v1"}, nil); len(ch) != 0 {
		t.Fatalf("same-host endpoint should produce no change: %#v", ch)
	}
	ch := ChangesFromDesired("hermes", snap, Desired{BaseURL: "https://other.example/v1"}, nil)
	if len(ch) != 1 || ch[0].Field != "baseUrl" || ch[0].From != "gateway.example" || ch[0].To != "other.example" {
		t.Fatalf("baseUrl change = %#v", ch)
	}
	if (Desired{BaseURL: "https://x"}).Empty() {
		t.Fatal("baseUrl should count as non-empty")
	}
	if got := Project(snap, "base-url"); got.BaseURL != "" {
		t.Fatalf("projection must not copy the host into baseUrl: %#v", got)
	}
}

func TestBaseURLChangesCompareFullEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name, current, want string
		changed             bool
	}{
		{"path", "https://gateway.example/old/v1", "https://gateway.example/new/v1", true},
		{"scheme", "http://gateway.example/v1", "https://gateway.example/v1", true},
		{"userinfo", "https://user:sk-test-aaa@gateway.example/v1", "https://user:sk-test-bbb@gateway.example/v1", true},
		{"query", "https://gateway.example/v1?token=sk-test-aaa", "https://gateway.example/v1?token=sk-test-bbb", true},
		{"empty endpoint", "", "https://gateway.example/v1", true},
		{"unchanged", "https://gateway.example/v1", "https://gateway.example/v1", false},
		{"omitted intent", "https://gateway.example/v1", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snap := Snapshot{Name: "hermes", BaseURLHost: "gateway.example"}
			changes := ChangesFromDesired("hermes", snap, Desired{BaseURL: tc.want}, &tc.current)
			if !tc.changed {
				if len(changes) != 0 {
					t.Fatalf("expected no changes: %#v", changes)
				}
				return
			}
			from := "gateway.example"
			if tc.current == "" {
				from = ""
			}
			if len(changes) != 1 || changes[0].Field != "baseUrl" || changes[0].From != from || changes[0].To != "gateway.example" {
				t.Fatal("expected one baseUrl change containing only endpoint hosts")
			}
		})
	}
}
