package edit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/oldwinter/harnessctl/internal/jsonc"
)

// Format names on-disk encodings we know how to mutate.
type Format string

const (
	TOML   Format = "toml"
	JSON   Format = "json"
	JSONC  Format = "jsonc"
	YAML   Format = "yaml"
	DotEnv Format = "dotenv"
)

// Caveats documents format-preservation limits (surfaced in README).
const Caveats = `
TOML: line-oriented key updates keep comments and unrelated tables; new keys are appended.
JSON: re-encoded with 2-space indent; key order may change. Strict JSON has no comments.
JSONC: comments and trailing commas are dropped on write (parsed, then emitted as JSON).
YAML: uses yaml.v3 nodes and keeps comments on untouched keys when the document is a mapping.
`

// Set updates a dotted path to a string value.
func Set(src []byte, format Format, path []string, value string) ([]byte, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("edit: empty path")
	}
	switch format {
	case TOML:
		return SetTOML(src, path, value)
	case JSON:
		return SetJSON(src, path, value)
	case JSONC:
		return SetJSONC(src, path, value)
	case YAML:
		return SetYAML(src, path, value)
	case DotEnv:
		return SetDotEnv(src, path[len(path)-1], value)
	default:
		return nil, fmt.Errorf("edit: unknown format %q", format)
	}
}

// SetJSON unmarshals an object, sets a nested string, and re-indents.
func SetJSON(src []byte, path []string, value string) ([]byte, error) {
	var root any
	if len(bytes.TrimSpace(src)) == 0 {
		root = map[string]any{}
	} else if err := json.Unmarshal(src, &root); err != nil {
		return nil, err
	}
	if err := setMapPath(&root, path, value); err != nil {
		return nil, err
	}
	return json.MarshalIndent(root, "", "  ")
}

// SetJSONC strips comments then writes JSON.
func SetJSONC(src []byte, path []string, value string) ([]byte, error) {
	cleaned := src
	if len(bytes.TrimSpace(src)) > 0 {
		var err error
		cleaned, err = jsonc.Strip(src)
		if err != nil {
			return nil, err
		}
	}
	return SetJSON(cleaned, path, value)
}

func setMapPath(root *any, path []string, value string) error {
	if *root == nil {
		*root = map[string]any{}
	}
	m, ok := (*root).(map[string]any)
	if !ok {
		return fmt.Errorf("edit: root is not an object")
	}
	cur := m
	for i, k := range path {
		if i == len(path)-1 {
			cur[k] = value
			return nil
		}
		next, ok := cur[k]
		if !ok || next == nil {
			nm := map[string]any{}
			cur[k] = nm
			cur = nm
			continue
		}
		nm, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("edit: %s is not an object", strings.Join(path[:i+1], "."))
		}
		cur = nm
	}
	return nil
}

// SetYAML sets a mapping path using yaml.Node (comments on other keys kept).
func SetYAML(src []byte, path []string, value string) ([]byte, error) {
	var doc yaml.Node
	if len(bytes.TrimSpace(src)) == 0 {
		src = []byte("{}\n")
	}
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, err
	}
	if err := setYAMLNode(&doc, path, value); err != nil {
		return nil, err
	}
	return yaml.Marshal(&doc)
}

func setYAMLNode(n *yaml.Node, path []string, value string) error {
	if n == nil {
		return fmt.Errorf("edit: nil yaml node")
	}
	if n.Kind == yaml.DocumentNode {
		if len(n.Content) == 0 {
			n.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
		}
		return setYAMLNode(n.Content[0], path, value)
	}
	if n.Kind != yaml.MappingNode {
		// Replace a scalar model: "foo" with a mapping when path is nested.
		if n.Kind == yaml.ScalarNode && len(path) > 0 {
			n.Kind = yaml.MappingNode
			n.Tag = "!!map"
			n.Value = ""
			n.Content = nil
		} else {
			return fmt.Errorf("edit: yaml node is not a mapping")
		}
	}
	key := path[0]
	for i := 0; i < len(n.Content)-1; i += 2 {
		if n.Content[i].Value != key {
			continue
		}
		val := n.Content[i+1]
		if len(path) == 1 {
			val.Kind = yaml.ScalarNode
			val.Tag = "!!str"
			val.Value = value
			return nil
		}
		return setYAMLNode(val, path[1:], value)
	}
	k := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	if len(path) == 1 {
		n.Content = append(n.Content, k, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value})
		return nil
	}
	child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	n.Content = append(n.Content, k, child)
	return setYAMLNode(child, path[1:], value)
}

// SetDotEnv sets KEY=value, preserving other lines and comments.
func SetDotEnv(src []byte, key, value string) ([]byte, error) {
	lines := splitKeep(src)
	found := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		k, _, ok := strings.Cut(trim, "=")
		if !ok {
			continue
		}
		if strings.TrimSpace(k) != key {
			continue
		}
		lines[i] = key + "=" + value
		found = true
		break
	}
	if !found {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, key+"="+value)
		} else {
			lines = append(lines[:len(lines)-0], key+"="+value)
		}
	}
	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return []byte(out), nil
}

func splitKeep(src []byte) []string {
	s := string(src)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return []string{}
	}
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

// SetTOML updates a key, keeping comments and unrelated tables.
// path is [table..., key]. A one-element path is a top-level key.
func SetTOML(src []byte, path []string, value string) ([]byte, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("edit: empty toml path")
	}
	quoted := quoteTOML(value)
	lines := splitKeep(src)
	table := path[:len(path)-1]
	key := path[len(path)-1]
	wantHeader := tomlHeader(table)

	type pos struct{ line, tableDepth int }
	current := ""
	found := false
	insertAt := -1
	headerLine := -1
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "#") {
			continue
		}
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") && !strings.HasPrefix(trim, "[[") {
			hdr := strings.TrimSpace(trim[1 : len(trim)-1])
			current = hdr
			if hdr == wantHeader {
				headerLine = i
			}
			continue
		}
		if current != wantHeader {
			continue
		}
		k, rest, ok := cutAssign(line)
		if !ok || k != key {
			continue
		}
		comment := trailingComment(rest)
		lines[i] = k + " = " + quoted + comment
		found = true
		break
	}
	if found {
		return joinTOML(lines), nil
	}
	newline := key + " = " + quoted
	if wantHeader == "" {
		// append before first table
		insertAt = len(lines)
		for i, line := range lines {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "[") {
				insertAt = i
				break
			}
		}
		lines = insertLine(lines, insertAt, newline)
		return joinTOML(lines), nil
	}
	if headerLine >= 0 {
		// insert after header (and any following blanks/comments/keys — at end of table)
		end := headerLine + 1
		for end < len(lines) {
			trim := strings.TrimSpace(lines[end])
			if strings.HasPrefix(trim, "[") {
				break
			}
			end++
		}
		lines = insertLine(lines, end, newline)
		return joinTOML(lines), nil
	}
	// create table at EOF
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
		lines = append(lines, "")
	}
	lines = append(lines, "["+wantHeader+"]", newline)
	return joinTOML(lines), nil
}

func tomlHeader(table []string) string {
	if len(table) == 0 {
		return ""
	}
	parts := make([]string, len(table))
	for i, p := range table {
		if needsTOMLQuote(p) {
			parts[i] = `"` + strings.ReplaceAll(p, `"`, `\"`) + `"`
		} else {
			parts[i] = p
		}
	}
	return strings.Join(parts, ".")
}

func needsTOMLQuote(s string) bool {
	if s == "" {
		return true
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return true
	}
	return false
}

func quoteTOML(v string) string {
	b, _ := json.Marshal(v) // JSON string quoting is TOML-compatible
	return string(b)
}

func cutAssign(line string) (key, rest string, ok bool) {
	code := line
	if i := strings.Index(line, "#"); i >= 0 {
		// only treat # as comment if not inside quotes — good enough for our files
		if !strings.Contains(line[:i], "\"") || strings.Count(line[:i], "\"")%2 == 0 {
			code = line[:i]
		}
	}
	eq := strings.Index(code, "=")
	if eq < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(code[:eq])
	rest = strings.TrimSpace(line[eq+1:])
	return key, rest, key != ""
}

func trailingComment(rest string) string {
	inQ := false
	for i := 0; i < len(rest); i++ {
		if rest[i] == '"' && (i == 0 || rest[i-1] != '\\') {
			inQ = !inQ
			continue
		}
		if rest[i] == '#' && !inQ {
			return " " + strings.TrimSpace(rest[i:])
		}
	}
	return ""
}

func insertLine(lines []string, at int, line string) []string {
	if at < 0 {
		at = 0
	}
	if at > len(lines) {
		at = len(lines)
	}
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:at]...)
	out = append(out, line)
	out = append(out, lines[at:]...)
	return out
}

func joinTOML(lines []string) []byte {
	out := strings.Join(lines, "\n")
	if !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return []byte(out)
}
