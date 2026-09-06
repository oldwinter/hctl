package adapters

import (
	"testing"

	"github.com/oldwinter/hctl/internal/adapters/stub"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

type bareAdapter struct{}

func (bareAdapter) Name() string                             { return "bare" }
func (bareAdapter) Aliases() []string                        { return nil }
func (bareAdapter) BinaryNames() []string                    { return nil }
func (bareAdapter) ConfigRelPaths() []string                 { return []string{".bare/x"} }
func (bareAdapter) Read(home string) (model.Snapshot, error) { return model.Snapshot{}, nil }

func TestReadOneFSBareAndScan(t *testing.T) {
	home := t.TempDir()
	snap, err := ReadOneFS(bareAdapter{}, fsx.Local{}, home)
	if err != nil || snap.Name != "bare" || len(snap.ConfigPaths) == 0 {
		t.Fatalf("%#v %v", snap, err)
	}
	a := stub.New("s", []string{}, []string{".s/c"})
	snap, err = ReadOne(a, home)
	if err != nil {
		t.Fatal(err)
	}
	_ = snap
	_, err = Scan(home)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ByName("claude")
	if err != nil {
		t.Fatal(err)
	}
}
