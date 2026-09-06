package secret

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"regexp"
	"strings"
)

// Fingerprint returns the first 8 hex characters of SHA-256(secret).
// Empty input yields an empty fingerprint.
func Fingerprint(secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])[:8]
}

// HostOf returns the host[:port] of a URL-like string. Userinfo and the rest
// of the path/query are discarded so credentials never leak via the host field.
func HostOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return u.Host
}

var (
	// Common vendor prefixes plus a generic long token. Never treat "sk-test"
	// documentation mentions as the only pattern.
	secretPattern = regexp.MustCompile(`(?i)\b(sk-[a-z0-9_-]{6,}|sk-ant-[a-z0-9_-]{8,}|sk-or-[a-z0-9_-]{8,}|sk-proj-[a-z0-9_-]{8,})\b`)
	longHex       = regexp.MustCompile(`\b[a-f0-9]{32,}\b`)
)

// LooksLikeSecret reports whether s appears to contain a raw API key or token.
func LooksLikeSecret(s string) bool {
	if secretPattern.MatchString(s) {
		return true
	}
	// Long hex that is not a short fingerprint (8 chars) or sha256: prefix label.
	return longHex.MatchString(s)
}

// Redact replaces secret-shaped substrings. Used as a last-line renderer guard.
func Redact(s string) string {
	s = secretPattern.ReplaceAllString(s, "[redacted]")
	s = longHex.ReplaceAllString(s, "[redacted-hex]")
	return s
}

// EnvRef reports whether value looks like a reference (env var / {env:X}) rather
// than an inline secret.
func EnvRef(value string) (name string, isRef bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	if strings.HasPrefix(value, "{env:") && strings.HasSuffix(value, "}") {
		return strings.TrimSuffix(strings.TrimPrefix(value, "{env:"), "}"), true
	}
	if strings.HasPrefix(value, "$") {
		return strings.TrimPrefix(value, "$"), true
	}
	// Bare env-var style names: ALL_CAPS with underscore.
	if envName.MatchString(value) && !strings.Contains(value, "://") && !LooksLikeSecret(value) {
		return value, true
	}
	return "", false
}

var envName = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,}$`)
