package stub

import (
	"github.com/oldwinter/harnessctl/internal/fsx"
	"github.com/oldwinter/harnessctl/internal/model"
)

// Adapter is a read-only placeholder for harnesses without a dedicated parser.
type Adapter struct {
	name     string
	bins     []string
	relPaths []string
}

func New(name string, bins, relPaths []string) Adapter {
	return Adapter{name: name, bins: bins, relPaths: relPaths}
}

func (a Adapter) Name() string             { return a.name }
func (a Adapter) Aliases() []string        { return nil }
func (a Adapter) BinaryNames() []string    { return a.bins }
func (a Adapter) ConfigRelPaths() []string { return a.relPaths }

func (a Adapter) Read(home string) (model.Snapshot, error) {
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	var paths []string
	found := false
	for _, rel := range a.relPaths {
		p := fsys.Join(home, rel)
		paths = append(paths, p)
		if fsx.ExistsAny(fsys, p) {
			found = true
		}
	}
	snap := model.Snapshot{
		Name:        a.name,
		ConfigPaths: paths,
		ConfigFound: found,
	}
	if found {
		snap.Notes = append(snap.Notes, "config path exists; dedicated read adapter lands in a later release")
	}
	return snap, nil
}
