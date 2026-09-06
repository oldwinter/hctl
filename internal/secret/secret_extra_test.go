package secret

import "testing"

func TestHostEnvLooks(t *testing.T) {
	if HostOf("") != "" || HostOf("not a url ::::") == "x" {
		_ = HostOf("::::")
	}
	if HostOf("api.example.com") == "" {
		t.Fatal("host")
	}
	if !LooksLikeSecret("sk-test-abcdef") {
		t.Fatal("looks")
	}
	name, ok := EnvRef("{env:FOO}")
	if !ok || name != "FOO" {
		t.Fatal(name, ok)
	}
	name, ok = EnvRef("$BAR")
	if !ok || name != "BAR" {
		t.Fatal(name, ok)
	}
	name, ok = EnvRef("OPENAI_API_KEY")
	if !ok || name != "OPENAI_API_KEY" {
		t.Fatal(name, ok)
	}
	if _, ok := EnvRef("sk-test-abcdef"); ok {
		t.Fatal("not ref")
	}
	if Fingerprint("") != "" {
		t.Fatal("fp")
	}
}
