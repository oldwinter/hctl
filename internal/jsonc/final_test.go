package jsonc

import "testing"

func TestUnmarshalStripError(t *testing.T) {
	var v any
	if err := Unmarshal([]byte(`"unterminated`), &v); err == nil {
		t.Fatal("expected")
	}
}
