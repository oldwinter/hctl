package desired

import "testing"

func TestParseErrors(t *testing.T) {
	if _, err := Parse([]byte("model = [[["), ".toml"); err == nil {
		t.Fatal("toml")
	}
	if _, err := Parse([]byte(": :\nbad"), ".yaml"); err == nil {
		t.Fatal("yaml")
	}
	if _, err := Parse([]byte(": :\nbad"), ".yml"); err == nil {
		t.Fatal("yml")
	}
}
