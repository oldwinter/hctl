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
