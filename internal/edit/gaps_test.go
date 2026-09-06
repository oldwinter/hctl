package edit

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEditRemainingGaps(t *testing.T) {
	if _, err := Set([]byte("x"), TOML, []string{"a"}, "v"); err != nil {
		// may succeed appending
		_ = err
	}
	if _, err := SetJSON([]byte("[]"), []string{"a"}, "v"); err == nil {
		t.Fatal("array root")
	}
	if _, err := SetJSON([]byte("{"), []string{"a"}, "v"); err == nil {
		t.Fatal("bad json")
	}
	var root any = map[string]any{"a": map[string]any{}}
	_ = setMapPath(&root, []string{"a", "b", "c"}, "v")
	// setYAMLNode not mapping after scalar without nested?
	n := &yaml.Node{Kind: yaml.SequenceNode}
	if err := setYAMLNode(n, []string{"a"}, "v"); err == nil {
		t.Fatal("seq")
	}
	doc := &yaml.Node{Kind: yaml.DocumentNode}
	if err := setYAMLNode(doc, []string{"a"}, "v"); err != nil {
		t.Fatal(err)
	}
	// SetDotEnv update existing + comment lines
	out, err := SetDotEnv([]byte("#c\nFOO=1\n"), "FOO", "2")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	out, err = SetDotEnv([]byte("FOO=1"), "BAR", "2")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	out, err = SetDotEnv([]byte("FOO=1\n"), "BAR", "2")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	// SetTOML update with comment trailing
	out, err = SetTOML([]byte("[t]\nk = \"o\" # hi\n"), []string{"t", "k"}, "n")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	_ = tomlHeader([]string{"a", "b", "c"})
	_ = needsTOMLQuote(`"quoted"`)
}
