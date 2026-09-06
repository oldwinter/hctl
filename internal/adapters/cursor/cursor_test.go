package cursor

import (
	"testing"

	"github.com/oldwinter/hctl/internal/testutil"
)

func TestReadHomeA(t *testing.T) {
	snap, err := (Adapter{}).Read(testutil.Testdata(t, "home-a"))
	if err != nil {
		t.Fatal(err)
	}
	if snap.DefaultModel != "gpt-5" {
		t.Fatalf("%q", snap.DefaultModel)
	}
}
