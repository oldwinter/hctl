package grok

import (
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/oldwinter/harnessctl/internal/edit"
	"github.com/oldwinter/harnessctl/internal/fsx"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads and writes ~/.grok/config.toml.
type Adapter struct{}

func (Adapter) Name() string          { return "grok" }
func (Adapter) Aliases() []string     { return []string{"grok-build"} }
func (Adapter) BinaryNames() []string { return []string{"grok"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".grok/config.toml"}
}

type file struct {
	Models map[string]any          `toml:"models"`
	Model  map[string]modelSection `toml:"model"`
}

type modelSection struct {
	Model      string `toml:"model"`
	BaseURL    string `toml:"base_url"`
	APIKey     string `toml:"api_key"`
	EnvKey     any    `toml:"env_key"`
	APIBackend string `toml:"api_backend"`
	Reasoning  string `toml:"reasoning_effort"`
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	path := fsys.Join(home, ".grok", "config.toml")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{path}}
	data, err := fsys.ReadFile(path)
	if err != nil {
		if fsx.IsNotExist(err) {
			return snap, nil
		}
		return snap, err
	}
	snap.ConfigFound = true
	var cfg file
	if err := toml.Unmarshal(data, &cfg); err != nil {
		snap.ParseError = err.Error()
		return snap, nil
	}
	def := stringFromMap(cfg.Models, "default")
	snap.DefaultModel = def
	if def != "" && cfg.Model != nil {
		if sec, ok := cfg.Model[def]; ok {
			applyModel(&snap, sec)
		}
	}
	if snap.Provider == "" {
		switch {
		case strings.Contains(snap.BaseURLHost, "x.ai"):
			snap.Provider = "xai"
		case snap.BaseURLHost != "":
			snap.Provider = "custom"
		default:
			snap.Provider = "xai"
		}
	}
	return snap, nil
}

func applyModel(snap *model.Snapshot, sec modelSection) {
	if sec.BaseURL != "" {
		snap.BaseURLHost = secret.HostOf(sec.BaseURL)
	}
	if sec.Reasoning != "" {
		snap.Effort = sec.Reasoning
	}
	if sec.APIKey != "" {
		snap.SecretFingerprint = secret.Fingerprint(sec.APIKey)
		snap.SecretPresent = true
	}
	if ref := firstEnvKey(sec.EnvKey); ref != "" {
		snap.SecretRef = ref
	}
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func firstEnvKey(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	path := fsys.Join(home, ".grok", "config.toml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return nil, err
	}
	if d.Model != "" {
		data, err = edit.SetTOML(data, []string{"models", "default"}, d.Model)
		if err != nil {
			return nil, err
		}
	}
	if d.SecretRef != "" {
		snap, _ := a.ReadFS(fsys, home)
		name := d.Model
		if name == "" {
			name = snap.DefaultModel
		}
		if name == "" {
			name = "default"
		}
		data, err = edit.SetTOML(data, []string{"model", name, "env_key"}, d.SecretRef)
		if err != nil {
			return nil, err
		}
	}
	// provider is inferred from host; stored only as a note via no-op
	if err := fsx.AtomicWrite(fsys, path, data, 0o600); err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func (a Adapter) PeekSecret(fsys fsx.FS, home string) (ref, value string, err error) {
	path := fsys.Join(home, ".grok", "config.toml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil || len(data) == 0 {
		return "", "", err
	}
	var cfg file
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return "", "", err
	}
	def := stringFromMap(cfg.Models, "default")
	if def == "" || cfg.Model == nil {
		return "", "", nil
	}
	sec := cfg.Model[def]
	return firstEnvKey(sec.EnvKey), sec.APIKey, nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	path := fsys.Join(home, ".grok", "config.toml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return err
	}
	snap, _ := a.ReadFS(fsys, home)
	name := snap.DefaultModel
	if name == "" {
		name = "default"
	}
	if value != "" {
		data, err = edit.SetTOML(data, []string{"model", name, "api_key"}, value)
		if err != nil {
			return err
		}
	}
	if ref != "" {
		data, err = edit.SetTOML(data, []string{"model", name, "env_key"}, ref)
		if err != nil {
			return err
		}
	}
	return fsx.AtomicWrite(fsys, path, data, 0o600)
}
