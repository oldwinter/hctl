package secret

import "testing"

func TestHostOfBadAndLooksHex(t *testing.T) {
	_ = HostOf("http://[::1") // may parse or not
	if !LooksLikeSecret("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa") {
		t.Fatal("hex")
	}
	if LooksLikeSecret("short") {
		t.Fatal("short")
	}
	_ = Redact("sk-ant-abcdefghijklmnop")
	_, ok := EnvRef("")
	if ok {
		t.Fatal("empty")
	}
	_, ok = EnvRef("http://X")
	if ok {
		t.Fatal("url")
	}
}
