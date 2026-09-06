package desired

import "testing"

func TestParseBothFail(t *testing.T) {
	// invalid for both parsers
	_, err := Parse([]byte("[[[[["), ".unknown")
	if err == nil {
		t.Fatal("expected")
	}
}
