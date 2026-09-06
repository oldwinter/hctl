package secret

import (
	"strings"
	"testing"
)

func TestFingerprintStableAndShort(t *testing.T) {
	a := Fingerprint("sk-test-aaa")
	b := Fingerprint("sk-test-bbb")
	if len(a) != 8 || len(b) != 8 {
		t.Fatalf("want 8 hex chars, got %q %q", a, b)
	}
	if a == b {
		t.Fatal("different secrets must not share a fingerprint")
	}
	if Fingerprint("sk-test-aaa") != a {
		t.Fatal("fingerprint must be deterministic")
	}
	if Fingerprint("") != "" {
		t.Fatal("empty secret has empty fingerprint")
	}
}

func TestHostOfStripsUserinfoAndPath(t *testing.T) {
	got := HostOf("https://user:sk-test-aaa@api.example.com:8443/v1/chat")
	if got != "api.example.com:8443" {
		t.Fatalf("host = %q", got)
	}
	if LooksLikeSecret(got) {
		t.Fatal("host must not look like a secret")
	}
}

func TestLooksLikeSecret(t *testing.T) {
	if !LooksLikeSecret("token=sk-test-aaa leftover") {
		t.Fatal("expected to detect sk-test-aaa")
	}
	if LooksLikeSecret("sha256:deadbeef") {
		t.Fatal("short hex fingerprint must not look like a secret")
	}
	if !LooksLikeSecret(strings.Repeat("ab", 20)) { // 40 hex chars
		t.Fatal("long hex should look like a secret")
	}
}

func TestEnvRef(t *testing.T) {
	name, ok := EnvRef("{env:OPENCODE_API_KEY}")
	if !ok || name != "OPENCODE_API_KEY" {
		t.Fatalf("got %q %v", name, ok)
	}
	name, ok = EnvRef("ANTHROPIC_AUTH_TOKEN")
	if !ok || name != "ANTHROPIC_AUTH_TOKEN" {
		t.Fatalf("got %q %v", name, ok)
	}
	if _, ok := EnvRef("sk-test-aaa"); ok {
		t.Fatal("inline key is not an env ref")
	}
}

func TestRedact(t *testing.T) {
	out := Redact("key sk-test-aaa end")
	if strings.Contains(out, "sk-test-aaa") {
		t.Fatalf("leaked: %s", out)
	}
}
