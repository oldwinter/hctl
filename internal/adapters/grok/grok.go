package grok

import (
	"sort"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
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

func (a Adapter) Read(fsys fsx.FS, home string) (model.Snapshot, error) {
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

func (Adapter) UnsupportedDesiredFields() []string { return []string{"provider"} }

func (a Adapter) ValidateDesired(fsys fsx.FS, home string, d model.Desired) error {
	if d.Provider != "" {
		return exitcode.Errorf(exitcode.Usage, "set provider is unsupported for grok (inferred from base_url); use set model")
	}
	if d.Model == "" {
		return nil
	}
	cfg, err := readGrokFile(fsys, home)
	if err != nil {
		return err
	}
	if cfg.Model != nil {
		if _, ok := cfg.Model[d.Model]; ok {
			return nil
		}
		old := stringFromMap(cfg.Models, "default")
		if old != "" {
			if _, ok := cfg.Model[old]; ok {
				return nil
			}
		}
	}
	names := make([]string, 0, len(cfg.Model))
	for name := range cfg.Model {
		names = append(names, name)
	}
	sort.Strings(names)
	have := strings.Join(names, ", ")
	if have == "" {
		have = "(none)"
	}
	return exitcode.Errorf(exitcode.Usage, "grok model %q has no [model.%q] table and no current model table to copy; have %s", d.Model, d.Model, have)
}

func readGrokFile(fsys fsx.FS, home string) (file, error) {
	data, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".grok", "config.toml"))
	if err != nil || len(data) == 0 {
		return file{}, err
	}
	var cfg file
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return file{}, exitcode.Errorf(exitcode.Parse, "grok config.toml is invalid")
	}
	return cfg, nil
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	if err := a.ValidateDesired(fsys, home, d); err != nil {
		return nil, err
	}
	path := fsys.Join(home, ".grok", "config.toml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return nil, err
	}
	if d.Model != "" {
		var cfg file
		_ = toml.Unmarshal(data, &cfg)
		exists := false
		if cfg.Model != nil {
			_, exists = cfg.Model[d.Model]
		}
		if !exists {
			old := stringFromMap(cfg.Models, "default")
			if old != "" && cfg.Model != nil {
				if src, ok := cfg.Model[old]; ok {
					data, err = copyGrokModel(data, d.Model, src)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		data, err = edit.SetTOML(data, []string{"models", "default"}, d.Model)
		if err != nil {
			return nil, err
		}
	}
	if d.SecretRef != "" {
		snap, _ := a.Read(fsys, home)
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
	snap, _ := a.Read(fsys, home)
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

func copyGrokModel(data []byte, name string, src modelSection) ([]byte, error) {
	var err error
	if src.Model != "" {
		data, err = edit.SetTOML(data, []string{"model", name, "model"}, src.Model)
		if err != nil {
			return nil, err
		}
	}
	if src.BaseURL != "" {
		data, err = edit.SetTOML(data, []string{"model", name, "base_url"}, src.BaseURL)
		if err != nil {
			return nil, err
		}
	}
	if src.APIKey != "" {
		data, err = edit.SetTOML(data, []string{"model", name, "api_key"}, src.APIKey)
		if err != nil {
			return nil, err
		}
	}
	if ref := firstEnvKey(src.EnvKey); ref != "" {
		data, err = edit.SetTOML(data, []string{"model", name, "env_key"}, ref)
		if err != nil {
			return nil, err
		}
	}
	if src.APIBackend != "" {
		data, err = edit.SetTOML(data, []string{"model", name, "api_backend"}, src.APIBackend)
		if err != nil {
			return nil, err
		}
	}
	if src.Reasoning != "" {
		data, err = edit.SetTOML(data, []string{"model", name, "reasoning_effort"}, src.Reasoning)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}
