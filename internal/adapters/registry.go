package adapters

import (
	"fmt"
	"strings"

	"github.com/oldwinter/hctl/internal/adapters/claude"
	"github.com/oldwinter/hctl/internal/adapters/codex"
	"github.com/oldwinter/hctl/internal/adapters/cursor"
	"github.com/oldwinter/hctl/internal/adapters/droid"
	"github.com/oldwinter/hctl/internal/adapters/grok"
	"github.com/oldwinter/hctl/internal/adapters/hermes"
	"github.com/oldwinter/hctl/internal/adapters/opencode"
	"github.com/oldwinter/hctl/internal/adapters/pi"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

// Adapter reads one harness's on-disk config into the unified snapshot.
type Adapter interface {
	Name() string
	Aliases() []string
	BinaryNames() []string
	ConfigRelPaths() []string
	Read(home string) (model.Snapshot, error)
}

// FSReader reads via an abstract filesystem (local or SSH).
type FSReader interface {
	ReadFS(fsys fsx.FS, home string) (model.Snapshot, error)
}

// All returns adapters in display order.
func All() []Adapter {
	return []Adapter{
		codex.Adapter{},
		claude.Adapter{},
		grok.Adapter{},
		hermes.Adapter{},
		opencode.Adapter{},
		pi.Adapter{},
		droid.Adapter{},
		cursor.Adapter{},
	}
}

// ByName resolves an official name or alias.
func ByName(name string) (Adapter, error) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, a := range All() {
		if strings.EqualFold(a.Name(), want) {
			return a, nil
		}
		for _, al := range a.Aliases() {
			if strings.EqualFold(al, want) {
				return a, nil
			}
		}
	}
	return nil, fmt.Errorf("unknown harness %q", name)
}

// Names returns official harness names.
func Names() []string {
	var out []string
	for _, a := range All() {
		out = append(out, a.Name())
	}
	return out
}

func Scan(home string) ([]model.Snapshot, error) {
	return ScanFS(fsx.Local{}, home)
}

func ScanFS(fsys fsx.FS, home string) ([]model.Snapshot, error) {
	home = strings.TrimSpace(home)
	if home == "" {
		return nil, fmt.Errorf("home is empty")
	}
	var out []model.Snapshot
	for _, a := range All() {
		snap, err := ReadOneFS(a, fsys, home)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, nil
}

func ReadOne(a Adapter, home string) (model.Snapshot, error) {
	return ReadOneFS(a, fsx.Local{}, home)
}

func ReadOneFS(a Adapter, fsys fsx.FS, home string) (model.Snapshot, error) {
	var (
		snap model.Snapshot
		err  error
	)
	if r, ok := a.(FSReader); ok {
		snap, err = r.ReadFS(fsys, home)
	} else {
		snap, err = a.Read(home)
	}
	if err != nil {
		return model.Snapshot{}, err
	}
	if snap.Name == "" {
		snap.Name = a.Name()
	}
	path, ver, ok := DetectBinaryFS(fsys, a.BinaryNames())
	snap.Installed = ok
	snap.InstalledPath = path
	if snap.Version == "" {
		snap.Version = ver
	}
	if len(snap.ConfigPaths) == 0 {
		for _, rel := range a.ConfigRelPaths() {
			snap.ConfigPaths = append(snap.ConfigPaths, fsys.Join(home, rel))
		}
	}
	if !snap.ConfigFound {
		for _, p := range snap.ConfigPaths {
			if fsx.Exists(fsys, p) {
				snap.ConfigFound = true
				break
			}
		}
	}
	return snap, nil
}
