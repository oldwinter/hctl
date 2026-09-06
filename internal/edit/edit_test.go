package edit

import (
	"strings"
	"testing"
)

func TestSetTOMLPreservesComments(t *testing.T) {
	src := []byte(`# keep
model = "old" # trailing
model_provider = "custom"

[models]
default = "x"

[model_providers.custom]
base_url = "https://api.example.com/v1" # host
`)
	out, err := SetTOML(src, []string{"model"}, "new-model")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `# keep`) || !strings.Contains(s, `# trailing`) {
		t.Fatal(s)
	}
	if !strings.Contains(s, `model = "new-model"`) {
		t.Fatal(s)
	}
	out, err = SetTOML(out, []string{"models", "default"}, "y")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `default = "y"`) {
		t.Fatal(string(out))
	}
	if !strings.Contains(string(out), `base_url = "https://api.example.com/v1"`) {
		t.Fatal(string(out))
	}
}

func TestSetTOMLCreatesTable(t *testing.T) {
	out, err := SetTOML(nil, []string{"models", "default"}, "grok-4.6")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "[models]") || !strings.Contains(string(out), `default = "grok-4.6"`) {
		t.Fatal(string(out))
	}
}

func TestSetJSONAndJSONC(t *testing.T) {
	src := []byte(`{
  // comment
  "model": "old",
  "env": { "A": "1" },
}`)
	out, err := SetJSONC(src, []string{"model"}, "new")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"model": "new"`) {
		t.Fatal(string(out))
	}
	if !strings.Contains(string(out), `"A": "1"`) {
		t.Fatal(string(out))
	}
}

func TestSetYAMLNested(t *testing.T) {
	src := []byte(`# header
model:
  default: old
  provider: custom
providers:
  custom:
    base_url: https://x.example
`)
	out, err := SetYAML(src, []string{"model", "default"}, "new")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "new") || !strings.Contains(s, "custom") {
		t.Fatal(s)
	}
}

func TestSetDotEnv(t *testing.T) {
	src := []byte("# keys\nOPENROUTER_API_KEY=sk-test-aaa\n")
	out, err := SetDotEnv(src, "OPENROUTER_API_KEY", "sk-test-bbb")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "# keys") {
		t.Fatal(string(out))
	}
	if !strings.Contains(string(out), "OPENROUTER_API_KEY=sk-test-bbb") {
		t.Fatal(string(out))
	}
}
