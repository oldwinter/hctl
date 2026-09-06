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
	ch := DiffAgainst(want, snaps)
	if len(ch) == 0 {
		t.Fatal("expected model change")
	}
}
