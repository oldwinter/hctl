package edit

import (
	"testing"
)

func TestSetYAMLViaSet(t *testing.T) {
	out, err := Set([]byte("a: 1\n"), YAML, []string{"a"}, "2")
	if err != nil || len(out) == 0 {
		t.Fatal(err)
	}
}

func TestSetMapPathEmptyReturn(t *testing.T) {
	var root any = map[string]any{}
	if err := setMapPath(&root, nil, "v"); err != nil {
		t.Fatal(err)
	}
}

func TestSetYAMLImplNodeErr(t *testing.T) {
	if _, err := setYAMLImpl([]byte("[]\n"), []string{"k"}, "v"); err == nil {
		t.Fatal("expected")
	}
}
