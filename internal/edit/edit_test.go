package edit

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestTOMLTableBoundaries(t *testing.T) {
	cases := []struct {
		name string
		src  string
		path []string
	}{
		{
			name: "commented-header",
			src:  "[model_providers.custom] # preserve\nenv_key = \"OLD_KEY\" # key comment\nbase_url = \"https://api.example.com\"\n",
			path: []string{"model_providers", "custom", "env_key"},
		},
		{
			name: "array-header",
			src:  "# preserve\n[[profiles]]\nmodel = \"profile-model\" # key comment\n",
			path: []string{"model"},
		},
		{
			name: "commented-header-insert",
			src:  "[model_providers.custom]# preserve\nbase_url = \"https://api.example.com\"\n[other] # next table\nenv_key = \"OTHER_KEY\" # key comment\n",
			path: []string{"model_providers", "custom", "env_key"},
		},
		{
			name: "commented-header-ends-root",
			src:  "[profile] # preserve\nmodel = \"profile-model\" # key comment\n",
			path: []string{"model"},
		},
		{
			name: "commented-header-ends-table",
			src:  "[models]\nkeep = true\n[other] # preserve\ndefault = \"other-model\" # key comment\n",
			path: []string{"models", "default"},
		},
		{
			name: "commented-array-headers",
			src:  "[[profiles]] # preserve\nmodel = \"first-profile\" # key comment\n[[profiles]]# second profile\nmodel = \"second-profile\"\n",
			path: []string{"model"},
		},
		{
			name: "array-header-ends-table",
			src:  "[models]\nkeep = true\n[[profiles]]\ndefault = \"profile-model\" # key comment\n",
			path: []string{"models", "default"},
		},
		{
			name: "table-after-array-header",
			src:  "[[profiles]] # preserve\nmodel = \"profile-model\"\n[models] # model settings\ndefault = \"old-model\" # key comment\n",
			path: []string{"models", "default"},
		},
		{
			name: "quoted-hash-in-header",
			src:  "[model_providers.\"cus#tom\"] # preserve\nenv_key = \"OLD_KEY\" # key comment\n",
			path: []string{"model_providers", "cus#tom", "env_key"},
		},
		{
			name: "escaped-quote-in-header",
			src: `[model_providers."cus\"#tom"] # preserve
env_key = "OLD_KEY" # key comment
`,
			path: []string{"model_providers", `cus"#tom`, "env_key"},
		},
		{
			name: "literal-quoted-header-ends-root",
			src:  "['cus#tom'] # preserve\nmodel = \"profile-model\" # key comment\n",
			path: []string{"model"},
		},
		{
			name: "escaped-backslash-in-header",
			src: `["custom\\"] # preserve " unmatched quote
model = "profile-model" # key comment
`,
			path: []string{"model"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var want map[string]any
			if err := toml.Unmarshal([]byte(tc.src), &want); err != nil {
				t.Fatalf("invalid fixture: %v", err)
			}
			table := want
			for _, key := range tc.path[:len(tc.path)-1] {
				table = table[key].(map[string]any)
			}
			table[tc.path[len(tc.path)-1]] = "new-value"
			out, err := SetTOML([]byte(tc.src), tc.path, "new-value")
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := toml.Unmarshal(out, &got); err != nil {
				t.Fatalf("invalid TOML after edit: %v\n%s", err, out)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %#v, want %#v", got, want)
			}
			for _, comment := range []string{"# preserve", "# key comment"} {
				if strings.Contains(tc.src, comment) && !strings.Contains(string(out), comment) {
					t.Errorf("lost comment %q\n%s", comment, out)
				}
			}
			for _, line := range strings.Split(tc.src, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "[") && !strings.Contains(string(out), line+"\n") {
					t.Errorf("changed header %q\n%s", line, out)
				}
			}
		})
	}
}

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
