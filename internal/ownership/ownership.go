package ownership

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
)

const (
	pointerOwner = "oldwinter/dotfiles"
	pointerPath  = ".config/harness/ownership.json"
)

// Options controls managed-path ownership checks for a write operation.
type Options struct {
	Manifest     string
	AllowManaged bool
}

type pointer struct {
	Version  int    `json:"version"`
	Owner    string `json:"owner"`
	Manifest string `json:"manifest"`
}

type manifest struct {
	Version int `json:"version"`
	Units   []struct {
		Dest string `json:"dest"`
	} `json:"units"`
}

// Check rejects writes to paths owned by the target's dotfiles manifest.
// A missing pointer and missing conventional manifest means the target is
// standalone. Present but invalid ownership data always fails closed.
func Check(fsys fsx.FS, home string, candidates []string, opts Options) error {
	managed, found, err := load(fsys, home, opts.Manifest)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	for _, candidate := range candidates {
		if _, ok := managed[candidate]; ok {
			if opts.AllowManaged {
				return nil
			}
			return exitcode.Errorf(exitcode.Usage, "managed harness destination is owned by %s; pass --allow-managed for a temporary override", pointerOwner)
		}
	}
	return nil
}

func load(fsys fsx.FS, home, explicit string) (map[string]struct{}, bool, error) {
	manifestPath := ""
	if explicit != "" {
		var err error
		manifestPath, err = resolveExplicit(fsys, home, explicit)
		if err != nil {
			return nil, false, err
		}
	} else {
		pointerData, err := fsx.ReadMaybe(fsys, fsys.Join(home, pointerPath))
		if err != nil {
			return nil, false, exitcode.Errorf(exitcode.Usage, "cannot read ownership pointer")
		}
		if pointerData != nil {
			var p pointer
			if err := json.Unmarshal(pointerData, &p); err != nil {
				return nil, false, exitcode.Errorf(exitcode.Usage, "invalid ownership pointer JSON")
			}
			if p.Version != 1 {
				return nil, false, exitcode.Errorf(exitcode.Usage, "unsupported ownership pointer version %d", p.Version)
			}
			if p.Owner != pointerOwner {
				return nil, false, exitcode.Errorf(exitcode.Usage, "unsupported ownership pointer owner")
			}
			if err := validateAbsolute(p.Manifest); err != nil {
				return nil, false, exitcode.Errorf(exitcode.Usage, "invalid ownership pointer manifest path")
			}
			manifestPath = p.Manifest
		} else {
			manifestPath = fsys.Join(home, "dotfiles", "harness", "manifest.json")
			data, err := fsx.ReadMaybe(fsys, manifestPath)
			if err != nil {
				return nil, false, exitcode.Errorf(exitcode.Usage, "cannot read conventional ownership manifest")
			}
			if data == nil {
				return nil, false, nil
			}
			return parseManifest(fsys, home, data)
		}
	}

	data, err := fsx.ReadMaybe(fsys, manifestPath)
	if err != nil {
		return nil, false, exitcode.Errorf(exitcode.Usage, "cannot read ownership manifest")
	}
	if data == nil {
		return nil, false, exitcode.Errorf(exitcode.Usage, "ownership manifest is missing")
	}
	return parseManifest(fsys, home, data)
}

func resolveExplicit(fsys fsx.FS, home, value string) (string, error) {
	switch {
	case strings.HasPrefix(value, "~/"):
		rel, err := validateDestination(value)
		if err != nil {
			return "", exitcode.Errorf(exitcode.Usage, "invalid --ownership-manifest path")
		}
		return joinRelative(fsys, home, rel), nil
	case isAbsolute(value):
		if err := validateAbsolute(value); err != nil {
			return "", exitcode.Errorf(exitcode.Usage, "invalid --ownership-manifest path")
		}
		return value, nil
	default:
		return "", exitcode.Errorf(exitcode.Usage, "--ownership-manifest must be an absolute or ~/ path")
	}
}

func parseManifest(fsys fsx.FS, home string, data []byte) (map[string]struct{}, bool, error) {
	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, false, exitcode.Errorf(exitcode.Usage, "invalid ownership manifest JSON")
	}
	if m.Version != 1 {
		return nil, false, exitcode.Errorf(exitcode.Usage, "unsupported ownership manifest version %d", m.Version)
	}
	if m.Units == nil {
		return nil, false, exitcode.Errorf(exitcode.Usage, "invalid ownership manifest: units must be an array")
	}
	managed := make(map[string]struct{}, len(m.Units))
	for i, unit := range m.Units {
		rel, err := validateDestination(unit.Dest)
		if err != nil {
			return nil, false, exitcode.Errorf(exitcode.Usage, "invalid ownership manifest destination at unit %d: %v", i, err)
		}
		managed[joinRelative(fsys, home, rel)] = struct{}{}
	}
	return managed, true, nil
}

func validateDestination(dest string) (string, error) {
	if !strings.HasPrefix(dest, "~/") {
		return "", fmt.Errorf("must begin with ~/")
	}
	rel := strings.TrimPrefix(dest, "~/")
	if rel == "" || path.IsAbs(rel) || strings.Contains(rel, "\\") || strings.ContainsRune(rel, '\x00') {
		return "", fmt.Errorf("must be a non-empty slash-separated relative path")
	}
	if cleaned := path.Clean(rel); cleaned != rel || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("must not contain traversal or non-canonical segments")
	}
	return rel, nil
}

func validateAbsolute(value string) error {
	if strings.ContainsRune(value, '\x00') {
		return fmt.Errorf("path must be absolute and canonical")
	}
	if path.IsAbs(value) && !strings.Contains(value, "\\") && path.Clean(value) == value {
		return nil
	}
	if !windowsAbsolute.MatchString(value) {
		return fmt.Errorf("path must be absolute and canonical")
	}
	normalized := strings.ReplaceAll(value, "\\", "/")
	parts := strings.Split(strings.TrimPrefix(normalized[2:], "/"), "/")
	if len(parts) == 0 {
		return fmt.Errorf("path must be absolute and canonical")
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("path must be absolute and canonical")
		}
	}
	return nil
}

func isAbsolute(value string) bool {
	return path.IsAbs(value) || windowsAbsolute.MatchString(value)
}

func joinRelative(fsys fsx.FS, home, rel string) string {
	parts := append([]string{home}, strings.Split(rel, "/")...)
	return fsys.Join(parts...)
}

var windowsAbsolute = regexp.MustCompile(`^[A-Za-z]:[\\/]`)
