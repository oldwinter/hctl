package render

import (
	"strings"
	"testing"
)

func TestJSONMarshalError(t *testing.T) {
	var buf strings.Builder
	if err := JSON(&buf, make(chan int)); err == nil {
		t.Fatal("expected marshal err")
	}
}
