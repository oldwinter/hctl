package edit

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFinalEditGaps(t *testing.T) {
	if _, err := Set(nil, TOML, []string{"a"}, "v"); err != nil {
		t.Fatal(err)
	}
	_ = tomlHeader([]string{"weird key", "ok"})
	_ = needsTOMLQuote("a.b")
	_ = needsTOMLQuote("simple")
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "other"},
			{Kind: yaml.ScalarNode, Value: "1"},
		},
	}}}
	if err := setYAMLNode(doc, []string{"a", "b"}, "v"); err != nil {
		t.Fatal(err)
	}
	out, err := SetTOML([]byte("x = 1\n"), []string{"tbl", "k"}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	out, err = SetTOML([]byte("[tbl]\na = 1\n[other]\nb = 2\n"), []string{"tbl", "new"}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	out, err = SetDotEnv(nil, "K", "v")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	out, err = SetDotEnv([]byte("noassign\n"), "K", "v")
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	if err := setYAMLNode(nil, []string{"a"}, "v"); err == nil {
		t.Fatal("nil")
	}
}
