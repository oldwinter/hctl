package edit

import "testing"

func TestSetUnknownAndYAMLFail(t *testing.T) {
	if _, err := Set(nil, Format("nope"), []string{"a"}, "v"); err == nil {
		t.Fatal("unknown")
	}
	if _, err := SetYAML([]byte(":\t:"), []string{"a"}, "v"); err == nil {
		t.Fatal("yaml")
	}
	var root any = map[string]any{"a": 1}
	if err := setMapPath(&root, []string{"a", "b"}, "v"); err == nil {
		t.Fatal("not object")
	}
}
