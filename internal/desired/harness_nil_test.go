package desired

import "testing"

func TestParseEmptyLeavesHarnessesInit(t *testing.T) {
	f, err := Parse([]byte("apiVersion = \"harnessctl/v1\"\n"), ".toml")
	if err != nil {
		t.Fatal(err)
	}
	if f.Harnesses == nil {
		t.Fatal("nil")
	}
	f2, err := Parse([]byte(""), "")
	if err != nil {
		t.Fatal(err)
	}
	_ = f2
}
