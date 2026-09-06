package desired

import (
	"testing"

	"github.com/oldwinter/hctl/internal/model"
)

func TestDiffAgainstAllFields(t *testing.T) {
	want := &model.DesiredFile{Harnesses: map[string]model.Desired{
		"codex": {Model: "m", Provider: "p", SecretRef: "K"},
	}}
	ch := DiffAgainst(want, []model.Snapshot{{Name: "codex"}})
	if len(ch) != 3 {
		t.Fatal(ch)
	}
	p := t.TempDir() + "/missing.toml"
	if _, err := Load(p); err == nil {
		t.Fatal("missing")
	}
	f, err := Parse([]byte("harnesses:\n  x: {}\n"), ".yaml")
	if err != nil {
		t.Fatal(err)
	}
	_ = f
}
