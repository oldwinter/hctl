package hermes

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/oldwinter/harnessctl/internal/edit"
	"github.com/oldwinter/harnessctl/internal/fsx"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads and writes ~/.hermes/config.yaml and fingerprints ~/.hermes/.env.
type Adapter struct{}

func (Adapter) Name() string          { return "hermes" }
func (Adapter) Aliases() []string     { return nil }
func (Adapter) BinaryNames() []string { return []string{"hermes"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".hermes/config.yaml", ".hermes/.env"}
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
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	cfgPath := fsys.Join(home, ".hermes", "config.yaml")
	envPath := fsys.Join(home, ".hermes", ".env")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{cfgPath, envPath}}
	data, err := fsys.ReadFile(cfgPath)
	if err != nil {
		if fsx.IsNotExist(err) {
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

	envKeys, _ := parseDotEnvBytes(mustRead(fsys, envPath))
	if snap.SecretRef != "" {
		if val := envKeys[snap.SecretRef]; val != "" {
			snap.SecretFingerprint = secret.Fingerprint(val)
			snap.SecretPresent = true
		}
	} else if len(envKeys) > 0 {
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

func mustRead(fsys fsx.FS, path string) []byte {
	data, _ := fsx.ReadMaybe(fsys, path)
	return data
}

func parseDotEnvBytes(data []byte) (map[string]string, error) {
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(v, `"'`)
	}
	return out, nil
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	path := fsys.Join(home, ".hermes", "config.yaml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return nil, err
	}
	if d.Model != "" {
		data, err = edit.SetYAML(data, []string{"model", "default"}, d.Model)
		if err != nil {
			return nil, err
		}
	}
	if d.Provider != "" {
		data, err = edit.SetYAML(data, []string{"model", "provider"}, d.Provider)
		if err != nil {
			return nil, err
		}
	}
	if d.SecretRef != "" {
		prov := d.Provider
		if prov == "" {
			snap, _ := a.ReadFS(fsys, home)
			prov = snap.Provider
		}
		if prov == "" || prov == "auto" {
			prov = "custom"
		}
		data, err = edit.SetYAML(data, []string{"providers", prov, "key_env"}, d.SecretRef)
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
	snap, err := a.ReadFS(fsys, home)
	if err != nil {
		return "", "", err
	}
	envPath := fsys.Join(home, ".hermes", ".env")
	keys, _ := parseDotEnvBytes(mustRead(fsys, envPath))
	if snap.SecretRef != "" {
		return snap.SecretRef, keys[snap.SecretRef], nil
	}
	for k, v := range keys {
		if v != "" {
			return k, v, nil
		}
	}
	return "", "", nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	envPath := fsys.Join(home, ".hermes", ".env")
	data, err := fsx.ReadMaybe(fsys, envPath)
	if err != nil {
		return err
	}
	key := ref
	if key == "" {
		key = "HERMES_API_KEY"
	}
	if value != "" {
		data, err = edit.SetDotEnv(data, key, value)
		if err != nil {
			return err
		}
	}
	return fsx.AtomicWrite(fsys, envPath, data, 0o600)
}
