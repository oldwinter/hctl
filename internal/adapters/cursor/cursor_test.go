package cursor

import (
	"testing"

	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/testutil"
)

func TestReadHomeA(t *testing.T) {
	snap, err := Adapter{}.Read(fsx.Local{}, testutil.Testdata(t, "home-a"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "gpt-5" {
		t.Fatalf("%q", snap.DefaultModel)
	}
}

func TestClassifyLoginPrefersNegativePhrases(t *testing.T) {
	cases := []struct {
		in       string
		logged   bool
		wantNote string
	}{
		{"not logged in", false, "cursor-agent status: not logged in"},
		{"unauthenticated", false, "cursor-agent status: not logged in"},
		{"logged in", true, "cursor-agent status: logged in"},
		{"authenticated", true, "cursor-agent status: logged in"},
	}
	for _, tc := range cases {
		got, note := classifyLogin(tc.in)
		if got != tc.logged || note != tc.wantNote {
			t.Fatalf("%q: logged=%v note=%q", tc.in, got, note)
		}
	}
}

func TestBinaryNamesDoNotIncludeEditor(t *testing.T) {
	for _, name := range (Adapter{}).BinaryNames() {
		if name == "cursor" {
			t.Fatal("BinaryNames must not include the GUI editor")
		}
	}
}
