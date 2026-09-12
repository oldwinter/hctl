package model

import "testing"

func TestChangesFromDesired(t *testing.T) {
	snap := Snapshot{Name: "codex", DefaultModel: "old", Provider: "custom", SecretRef: "OLD_KEY"}
	d := Desired{Model: "new", Provider: "custom", SecretRef: "NEW_KEY"}
	ch := ChangesFromDesired("codex", snap, d)
	if len(ch) != 2 {
		t.Fatalf("changes = %#v", ch)
	}
	if ch[0].Field != "model" || ch[0].From != "old" || ch[0].To != "new" {
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
}
