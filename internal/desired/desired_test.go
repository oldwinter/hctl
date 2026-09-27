package desired

import (
	"testing"

	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestLoadTOML(t *testing.T) {
	f, err := Load(testutil.Testdata(t, "desired.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if f.Harnesses["codex"].Model != "o4-mini" {
		t.Fatalf("%#v", f.Harnesses["codex"])
	}
}

func TestDiffAgainst(t *testing.T) {
	want, err := Load(testutil.Testdata(t, "desired.toml"))
	if err != nil {
		t.Fatal(err)
	}
	snaps := []model.Snapshot{{Name: "codex", DefaultModel: "gpt-5.2-codex", Provider: "custom"}}
	ch := DiffAgainst(want, snaps, nil)
	if len(ch) == 0 {
		t.Fatal("expected model change")
	}
	same := model.ChangesFromDesired("codex", snaps[0], want.Harnesses["codex"], nil)
	if len(same) == 0 {
		t.Fatal("shared table should produce the same codex model change")
	}
	var got model.Change
	for _, c := range ch {
		if c.Harness == "codex" && c.Field == "model" {
			got = c
			break
		}
	}
	if got != same[0] {
		t.Fatalf("diff vs table: %#v vs %#v", got, same[0])
	}
}

func TestDiffAgainstFullEndpoint(t *testing.T) {
	want := &model.DesiredFile{Harnesses: map[string]model.Desired{
		"hermes": {BaseURL: "https://gateway.example/new/v1"},
	}}
	snaps := []model.Snapshot{{Name: "hermes", BaseURLHost: "gateway.example"}}
	for _, current := range []string{"https://gateway.example/old/v1", "http://gateway.example/new/v1", ""} {
		changes := DiffAgainst(want, snaps, map[string]string{"hermes": current})
		if len(changes) != 1 || changes[0].Field != "baseUrl" {
			t.Fatalf("current=%q changes=%#v", current, changes)
		}
	}
	changes := DiffAgainst(want, snaps, map[string]string{"hermes": want.Harnesses["hermes"].BaseURL})
	if len(changes) != 0 {
		t.Fatalf("unchanged endpoint should have no diff: %#v", changes)
	}
}
