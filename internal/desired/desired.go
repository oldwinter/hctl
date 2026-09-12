package desired

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"

	"github.com/oldwinter/hctl/internal/model"
)

// Load reads a desired-state file (TOML or YAML, by extension or content).
func Load(path string) (*model.DesiredFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data, filepath.Ext(path))
}

func Parse(data []byte, ext string) (*model.DesiredFile, error) {
	var f model.DesiredFile
	ext = strings.ToLower(ext)
	switch ext {
	case ".toml":
		if err := toml.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("desired toml: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("desired yaml: %w", err)
		}
	default:
		if err := toml.Unmarshal(data, &f); err != nil {
			if err2 := yaml.Unmarshal(data, &f); err2 != nil {
				return nil, fmt.Errorf("desired: not toml (%v) or yaml (%v)", err, err2)
			}
		}
	}
	if f.Harnesses == nil {
		f.Harnesses = map[string]model.Desired{}
	}
	if f.APIVersion == "" {
		f.APIVersion = "harnessctl/v1"
	}
	if f.Kind == "" {
		f.Kind = "DesiredState"
	}
	return &f, nil
}

// DiffAgainst returns changes from current snapshots to desired using the
// shared field table. Unknown harness names are still compared against an
// empty snapshot; callers that must fail closed (diff -f, apply) use ByName.
func DiffAgainst(want *model.DesiredFile, snaps []model.Snapshot) []model.Change {
	byName := map[string]model.Snapshot{}
	for _, s := range snaps {
		byName[s.Name] = s
	}
	names := make([]string, 0, len(want.Harnesses))
	for name := range want.Harnesses {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []model.Change
	for _, name := range names {
		out = append(out, model.ChangesFromDesired(name, byName[name], want.Harnesses[name])...)
	}
	return out
}
