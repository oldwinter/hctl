package jsonc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode"
)

// Unmarshal parses JSONC (JSON with comments and trailing commas).
func Unmarshal(data []byte, v any) error {
	cleaned, err := Strip(data)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(cleaned, v); err != nil {
		return fmt.Errorf("jsonc: %w", err)
	}
	return nil
}

// Strip removes // and /* */ comments and trailing commas.
func Strip(in []byte) ([]byte, error) {
	var out bytes.Buffer
	out.Grow(len(in))

	const (
		normal = iota
		squote
		dquote
		lineComment
		blockComment
	)
	state := normal
	for i := 0; i < len(in); i++ {
		c := in[i]
		next := byte(0)
		if i+1 < len(in) {
			next = in[i+1]
		}
		switch state {
		case normal:
			if c == '/' && next == '/' {
				state = lineComment
				i++
				continue
			}
			if c == '/' && next == '*' {
				state = blockComment
				i++
				continue
			}
			if c == '"' {
				state = dquote
				out.WriteByte(c)
				continue
			}
			if c == '\'' {
				state = squote
				out.WriteByte(c)
				continue
			}
			if c == ',' {
				j := i + 1
				for j < len(in) && unicode.IsSpace(rune(in[j])) {
					j++
				}
				if j < len(in) && (in[j] == '}' || in[j] == ']') {
					continue
				}
			}
			out.WriteByte(c)
		case dquote:
			out.WriteByte(c)
			if c == '\\' && i+1 < len(in) {
				out.WriteByte(in[i+1])
				i++
				continue
			}
			if c == '"' {
				state = normal
			}
		case squote:
			out.WriteByte(c)
			if c == '\\' && i+1 < len(in) {
				out.WriteByte(in[i+1])
				i++
				continue
			}
			if c == '\'' {
				state = normal
			}
		case lineComment:
			if c == '\n' {
				out.WriteByte(c)
				state = normal
			}
		case blockComment:
			if c == '*' && next == '/' {
				state = normal
				i++
			}
		}
	}
	if state == dquote || state == squote {
		return nil, fmt.Errorf("jsonc: unterminated string")
	}
	if state == blockComment {
		return nil, fmt.Errorf("jsonc: unterminated block comment")
	}
	return out.Bytes(), nil
}
