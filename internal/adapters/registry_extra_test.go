package adapters

import (
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestByNameNamesReadOneDetect(t *testing.T) {
	a, err := ByName("openai-codex")
	if err != nil || a.Name() != "codex" {
		t.Fatalf("%v %v", a, err)
	}
	if _, err := ByName("nope"); err == nil {
		t.Fatal("expected")
	}
	names := Names()
	if len(names) < 5 {
		t.Fatal(names)
	}
	home := testutil.Testdata(t, "home-a")
	snap, err := ReadOne(codex.Adapter{}, home)
	if err != nil || snap.Name != "codex" {
		t.Fatalf("%#v %v", snap, err)
	}
	path, ver, ok := DetectBinary([]string{"go", "this-binary-does-not-exist-xyz"})
	if !ok || path == "" {
		t.Fatalf("%q %q %v", path, ver, ok)
	}
	_ = truncate("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz", 10)
	if truncate("hi", 10) != "hi" {
		t.Fatal("truncate")
	}
	// DetectBinaryFS without LookPath
	type bare struct{ fsx.Local }
	// Local has LookPath — use a wrapper without it by embedding only FS methods via SSH without Run LookPath path
	_, _, ok = DetectBinaryFS(struct{ fsx.FS }{fsx.Local{}}, []string{"go"})
	if ok {
		t.Fatal("expected no lookpath")
	}
	s := fsx.SSH{Target: "u@h", Run: func(stdin []byte, name string, args ...string) ([]byte, error) {
		return []byte("/bin/codex\n"), nil
	}}
	path, ver, ok = DetectBinaryFS(s, []string{"codex"})
	if !ok || path == "" || ver != "" {
		t.Fatalf("%q %q %v", path, ver, ok)
	}
	if _, err := ScanFS(fsx.Local{}, ""); err == nil {
		t.Fatal("empty home")
	}
}
