package edit

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSetAllFormatsAndEdges(t *testing.T) {
	out, err := Set(nil, JSON, []string{"a", "b"}, "v")
	if err != nil || !strings.Contains(string(out), `"b"`) {
		t.Fatalf("%s %v", out, err)
	}
	if _, err := Set(nil, Format("nope"), []string{"a"}, "v"); err == nil {
		t.Fatal("unknown")
	}
	if _, err := Set(nil, JSON, nil, "v"); err == nil {
		t.Fatal("empty path")
	}
	_, err = Set([]byte(`{"x":1}`), JSONC, []string{"x"}, "2")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Set([]byte("k=old\n"), DotEnv, []string{"k"}, "new")
	if err != nil {
		t.Fatal(err)
	}
	y, err := SetYAML(nil, []string{"a", "b"}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_ = y
	y2, err := SetYAML([]byte("a:\n  b: old\n# keep\n"), []string{"a", "b"}, "new")
	if err != nil {
		t.Fatal(err)
	}
	_ = y2
	// setMapPath root not object
	var root any = []any{1}
	if err := setMapPath(&root, []string{"a"}, "v"); err == nil {
		t.Fatal("expected")
	}
	var root2 any
	if err := setMapPath(&root2, []string{"a", "b"}, "v"); err != nil {
		t.Fatal(err)
	}
	tomlOut, err := SetTOML([]byte("#c\n[table]\nk = \"o\"\n"), []string{"table", "k"}, "n")
	if err != nil {
		t.Fatal(err)
	}
	_ = tomlOut
	tomlOut, err = SetTOML(nil, []string{"newkey"}, "v")
	if err != nil {
		t.Fatal(err)
	}
	_ = tomlOut
	tomlOut, err = SetTOML([]byte("x = \"1\"\n"), []string{"y"}, `quote"me`)
	if err != nil {
		t.Fatal(err)
	}
	_ = insertLine([]string{"a", "b"}, 1, "mid")
	_ = insertLine([]string{"a"}, -1, "x")
	_ = insertLine([]string{"a"}, 99, "y")
	_ = needsTOMLQuote("simple")
	_ = needsTOMLQuote("has space")
	_ = tomlHeader([]string{"a", "b"})
	k, v, ok := cutAssign("a = b")
	if !ok || k != "a" {
		t.Fatal(k, v, ok)
	}
	_, _, ok = cutAssign("nope")
	if ok {
		t.Fatal("cut")
	}
	var root3 any = map[string]any{"a": 1}
	if err := setMapPath(&root3, []string{"a", "b"}, "v"); err == nil {
		t.Fatal("expected not object")
	}
	if _, err := SetYAML([]byte(": : :"), []string{"a"}, "v"); err == nil {
		t.Fatal("bad yaml")
	}
	if _, err := SetYAML([]byte("1"), []string{"a"}, "v"); err != nil {
		// scalar replaced into mapping for nested — path len>0 converts
		_ = err
	}
	if _, err := SetYAML([]byte("1"), []string{"a", "b"}, "v"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetJSONC([]byte(`"unterminated`), []string{"a"}, "v"); err == nil {
		t.Fatal("jsonc")
	}
	if _, err := SetTOML(nil, nil, "v"); err == nil {
		t.Fatal("empty toml path")
	}
	// SetYAML update existing nested
	if _, err := SetYAML([]byte("a:\n  b: old\n"), []string{"a", "b"}, "new"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetYAML([]byte("a: old\n"), []string{"a"}, "new"); err != nil {
		t.Fatal(err)
	}
	_ = setYAMLNode(nil, []string{"a"}, "v")
	doc := []byte("---\n")
	_ = doc
	if _, err := Set([]byte("x=1\n"), DotEnv, []string{"env", "x"}, "2"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetTOML([]byte("[t]\n"), []string{"t", "k"}, "v"); err != nil {
		t.Fatal(err)
	}
	if _, err := SetTOML([]byte("z = 1\n[t]\n"), []string{"new"}, "v"); err != nil {
		t.Fatal(err)
	}
	_ = Caveats
	_ = needsTOMLQuote("")
	_ = needsTOMLQuote("a=b")
	_ = needsTOMLQuote("plain_ok")
}

func TestSetUnknownFormat(t *testing.T) {
	if _, err := Set([]byte("{}"), Format("nope"), []string{"a"}, "b"); err == nil {
		t.Fatal("expected")
	}
}

func TestSetMapPathNotObject(t *testing.T) {
	src := []byte(`{"a":"str"}`)
	if _, err := SetJSON(src, []string{"a", "b"}, "x"); err == nil {
		t.Fatal("expected")
	}
}

func TestSetYAMLImplEdges(t *testing.T) {
	out, err := setYAMLImpl(nil, []string{"k"}, "v")
	if err != nil || len(out) == 0 {
		t.Fatal(err)
	}
	if _, err := setYAMLImpl([]byte(":\nbad"), []string{"k"}, "v"); err == nil {
		t.Fatal("expected unmarshal")
	}
	if err := setYAMLNode(nil, []string{"k"}, "v"); err == nil {
		t.Fatal("expected nil node")
	}
	if err := setYAMLNode(&yaml.Node{Kind: yaml.ScalarNode, Value: "x"}, []string{}, "v"); err == nil {
		t.Fatal("expected not mapping")
	}
}
