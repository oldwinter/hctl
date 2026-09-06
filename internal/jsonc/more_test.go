package jsonc

import "testing"

func TestStripCommentsAndComma(t *testing.T) {
	out, err := Strip([]byte("{\n  // c\n  \"a\": 1,\n  /*b*/\n}\n"))
	if err != nil {
		t.Fatal(err)
	}
	_ = out
	var v any
	if err := Unmarshal([]byte("{\n\"a\":1,\n}"), &v); err != nil {
		t.Fatal(err)
	}
}
