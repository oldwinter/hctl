package jsonc

import (
	"testing"
)

func TestStripCommentsAndTrailingCommas(t *testing.T) {
	in := []byte(`{
  // comment
  "model": "acme/gpt-5",
  "provider": {
    "acme": {
      "options": {
        "baseURL": "https://api.acme.example/v1",
        "apiKey": "sk-test-aaa",
      },
    },
  },
}
`)
	var v map[string]any
	if err := Unmarshal(in, &v); err != nil {
		t.Fatal(err)
	}
	if v["model"] != "acme/gpt-5" {
		t.Fatalf("model = %#v", v["model"])
	}
	prov := v["provider"].(map[string]any)
	acme := prov["acme"].(map[string]any)
	opts := acme["options"].(map[string]any)
	if opts["apiKey"] != "sk-test-aaa" {
		t.Fatalf("apiKey = %#v", opts["apiKey"])
	}
}

func TestDoesNotStripInsideStrings(t *testing.T) {
	in := []byte(`{"note": "http://not-a-comment", "ok": true}`)
	var v map[string]any
	if err := Unmarshal(in, &v); err != nil {
		t.Fatal(err)
	}
	if v["note"] != "http://not-a-comment" {
		t.Fatalf("note = %#v", v["note"])
	}
}

func TestHasCommentsOrTrailingCommas(t *testing.T) {
	if !HasCommentsOrTrailingCommas([]byte("{\n  // c\n  \"a\": 1,\n}")) {
		t.Fatal("expected comments")
	}
	if HasCommentsOrTrailingCommas([]byte(`{"a":1}`)) {
		t.Fatal("plain json")
	}
}

func TestStripTrailingCommaBeforeComment(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"line comment", "{\"a\": 1, // c\n}"},
		{"block comment", `{"a": 1, /* c */ }`},
		{"stacked line comments", "{\"a\": 1, // a\n // b\n}"},
		{"array", "[1, 2, // c\n]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var v any
			if err := Unmarshal([]byte(tc.in), &v); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCommaBeforeCommentAndValueSurvives(t *testing.T) {
	in := []byte("{\"a\": 1, // c\n \"b\": 2}")
	var v map[string]any
	if err := Unmarshal(in, &v); err != nil {
		t.Fatal(err)
	}
	if v["b"].(float64) != 2 {
		t.Fatalf("%#v", v)
	}
}

func TestBlockComment(t *testing.T) {
	in := []byte(`{"a": 1, /* skip */ "b": 2}`)
	var v map[string]any
	if err := Unmarshal(in, &v); err != nil {
		t.Fatal(err)
	}
	if v["b"].(float64) != 2 {
		t.Fatalf("%#v", v)
	}
}

func TestBlockCommentCannotJoinNumberTokens(t *testing.T) {
	var v map[string]any
	if err := Unmarshal([]byte(`{"value": 1/* comment */2}`), &v); err == nil {
		t.Fatalf("invalid JSONC parsed as %#v", v)
	}
}
