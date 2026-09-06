package grok

import (
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads ~/.grok/config.toml.
type Adapter struct{}

func (Adapter) Name() string          { return "grok" }
func (Adapter) Aliases() []string     { return []string{"grok-build"} }
func (Adapter) BinaryNames() []string { return []string{"grok"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{filepath.Join(".grok", "config.toml")}
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
	path := filepath.Join(home, ".grok", "config.toml")
	snap := model.Snapshot{
		Name:        a.Name(),
		ConfigPaths: []string{path},
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
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
	if def != "" {
		if sec, ok := cfg.Model[def]; ok {
			applyModel(&snap, sec)
		} else {
			// quoted keys like grok-4.6 should already match; try raw lookup
			for name, sec := range cfg.Model {
				if name == def {
					applyModel(&snap, sec)
					break
				}
			}
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
