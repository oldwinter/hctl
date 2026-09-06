package hermes

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads ~/.hermes/config.yaml and fingerprints keys in ~/.hermes/.env.
type Adapter struct{}

func (Adapter) Name() string          { return "hermes" }
func (Adapter) Aliases() []string     { return nil }
func (Adapter) BinaryNames() []string { return []string{"hermes"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{
		filepath.Join(".hermes", "config.yaml"),
		filepath.Join(".hermes", ".env"),
	}
}

type file struct {
	Model     any                    `yaml:"model"`
	Providers map[string]providerSec `yaml:"providers"`
}

type providerSec struct {
	BaseURL string `yaml:"base_url"`
	KeyEnv  string `yaml:"key_env"`
	APIKey  string `yaml:"api_key"`
}

type modelObj struct {
	Default  string `yaml:"default"`
	Provider string `yaml:"provider"`
	BaseURL  string `yaml:"base_url"`
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	cfgPath := filepath.Join(home, ".hermes", "config.yaml")
	envPath := filepath.Join(home, ".hermes", ".env")
	snap := model.Snapshot{
		Name:        a.Name(),
		ConfigPaths: []string{cfgPath, envPath},
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return snap, nil
		}
		return snap, err
	}
	snap.ConfigFound = true
	var cfg file
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		snap.ParseError = err.Error()
		return snap, nil
	}
	switch m := cfg.Model.(type) {
	case string:
		snap.DefaultModel = m
		if m == "" {
			snap.Notes = append(snap.Notes, "empty model sentinel — onboarding needed")
		}
	case map[string]any:
		var obj modelObj
		raw, _ := yaml.Marshal(m)
		_ = yaml.Unmarshal(raw, &obj)
		snap.DefaultModel = obj.Default
		snap.Provider = obj.Provider
		if obj.BaseURL != "" {
			snap.BaseURLHost = secret.HostOf(obj.BaseURL)
		}
	}

	if snap.Provider != "" && cfg.Providers != nil {
		if p, ok := cfg.Providers[snap.Provider]; ok {
			applyProvider(&snap, p)
		}
	} else if len(cfg.Providers) == 1 {
		for name, p := range cfg.Providers {
			if snap.Provider == "" {
				snap.Provider = name
			}
			applyProvider(&snap, p)
		}
	}

	envKeys, _ := parseDotEnv(envPath)
	if snap.SecretRef != "" {
		if val, ok := envKeys[snap.SecretRef]; ok && val != "" {
			snap.SecretFingerprint = secret.Fingerprint(val)
			snap.SecretPresent = true
		}
	} else if len(envKeys) > 0 {
		// Prefer a well-known key; otherwise first secret-looking value.
		for _, k := range []string{"OPENROUTER_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY", "HERMES_API_KEY"} {
			if val := envKeys[k]; val != "" {
				snap.SecretRef = k
				snap.SecretFingerprint = secret.Fingerprint(val)
				snap.SecretPresent = true
				break
			}
		}
	}
	if snap.Provider == "" {
		snap.Provider = "auto"
	}
	return snap, nil
}

func applyProvider(snap *model.Snapshot, p providerSec) {
	if p.BaseURL != "" {
		snap.BaseURLHost = secret.HostOf(p.BaseURL)
	}
	if p.KeyEnv != "" {
		snap.SecretRef = p.KeyEnv
	}
	if p.APIKey != "" {
		snap.SecretFingerprint = secret.Fingerprint(p.APIKey)
		snap.SecretPresent = true
	}
}

func parseDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"'`)
		out[strings.TrimSpace(k)] = v
	}
	return out, sc.Err()
}
