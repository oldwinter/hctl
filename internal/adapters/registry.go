package adapters

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oldwinter/harnessctl/internal/adapters/claude"
	"github.com/oldwinter/harnessctl/internal/adapters/codex"
	"github.com/oldwinter/harnessctl/internal/adapters/grok"
	"github.com/oldwinter/harnessctl/internal/adapters/hermes"
	"github.com/oldwinter/harnessctl/internal/adapters/opencode"
	"github.com/oldwinter/harnessctl/internal/adapters/stub"
	"github.com/oldwinter/harnessctl/internal/model"
)

// Adapter reads one harness's on-disk config into the unified snapshot.
type Adapter interface {
	Name() string
	Aliases() []string
	BinaryNames() []string
	ConfigRelPaths() []string
	Read(home string) (model.Snapshot, error)
}

// All returns adapters in display order.
func All() []Adapter {
	return []Adapter{
		codex.Adapter{},
		claude.Adapter{},
		grok.Adapter{},
		hermes.Adapter{},
		opencode.Adapter{},
		stub.New("pi", []string{"pi"}, []string{".pi", ".config/pi"}),
		stub.New("droid", []string{"droid"}, []string{".droid", ".factory"}),
		stub.New("cursor-agent", []string{"cursor-agent", "cursor"}, []string{".cursor"}),
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

// Scan reads every adapter under home.
func Scan(home string) ([]model.Snapshot, error) {
	home = strings.TrimSpace(home)
	if home == "" {
		return nil, fmt.Errorf("home is empty")
	}
	var out []model.Snapshot
	for _, a := range All() {
		snap, err := ReadOne(a, home)
		if err != nil {
			return nil, err
		}
		out = append(out, snap)
	}
	return out, nil
}

// ReadOne fills install metadata then delegates to the adapter parser.
func ReadOne(a Adapter, home string) (model.Snapshot, error) {
	snap, err := a.Read(home)
	if err != nil {
		return model.Snapshot{}, err
	}
	if snap.Name == "" {
		snap.Name = a.Name()
	}
	path, ver, ok := DetectBinary(a.BinaryNames())
	snap.Installed = ok
	snap.InstalledPath = path
	snap.Version = ver

	if len(snap.ConfigPaths) == 0 {
		for _, rel := range a.ConfigRelPaths() {
			snap.ConfigPaths = append(snap.ConfigPaths, filepath.Join(home, rel))
		}
	}
	if !snap.ConfigFound {
		for _, p := range snap.ConfigPaths {
			if fileExists(p) {
				snap.ConfigFound = true
				break
			}
		}
	}
	return snap, nil
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
