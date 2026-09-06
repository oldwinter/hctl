package jsonc

import "testing"

func TestStripEdges(t *testing.T) {
	var v map[string]any
	if err := Unmarshal([]byte(`{"a":1,}`), &v); err != nil {
		t.Fatal(err)
	}
	_, err := Strip([]byte(`"unterminated`))
	if err == nil {
		t.Fatal("unterminated")
	}
	_, err = Strip([]byte(`/* block`))
	if err == nil {
		t.Fatal("block")
	}
	out, err := Strip([]byte(`{'a':1}`))
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	out, err = Strip([]byte(`"a\"b"`))
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	out, err = Strip([]byte(`'a\'b'`))
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	if err := Unmarshal([]byte(`{`), &v); err == nil {
		t.Fatal("bad json")
	}
}
