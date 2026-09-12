package hermes

import (
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
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
	Model           any                    `yaml:"model"`
	Providers       map[string]providerSec `yaml:"providers"`
	CustomProviders []customProvider       `yaml:"custom_providers"`
}

type providerSec struct {
	BaseURL string `yaml:"base_url"`
	KeyEnv  string `yaml:"key_env"`
	APIKey  string `yaml:"api_key"`
	API     string `yaml:"api"`
}

type customProvider struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type modelObj struct {
	Default  string `yaml:"default"`
	Provider string `yaml:"provider"`
	BaseURL  string `yaml:"base_url"`
}

func (a Adapter) Read(fsys fsx.FS, home string) (model.Snapshot, error) {
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

	providerID := strings.TrimPrefix(snap.Provider, "custom:")
	if providerID != "" && cfg.Providers != nil {
		if p, ok := cfg.Providers[providerID]; ok {
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
	activeEndpoint := ""
	if p, ok := cfg.Providers[providerID]; ok {
		activeEndpoint = firstNonEmpty(p.BaseURL, p.API)
	}
	if p, ambiguous := activeCustomProvider(cfg, providerID, activeEndpoint); p != nil {
		if p.BaseURL != "" {
			snap.BaseURLHost = secret.HostOf(p.BaseURL)
		}
		if p.APIKey != "" {
			snap.SecretFingerprint = secret.Fingerprint(p.APIKey)
			snap.SecretPresent = true
		}
	} else if ambiguous {
		snap.Notes = append(snap.Notes, "ambiguous Hermes custom provider endpoint; secret not selected")
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
	if endpoint := firstNonEmpty(p.BaseURL, p.API); endpoint != "" {
		snap.BaseURLHost = secret.HostOf(endpoint)
	}
	if p.KeyEnv != "" {
		snap.SecretRef = p.KeyEnv
	}
	if p.APIKey != "" {
		snap.SecretFingerprint = secret.Fingerprint(p.APIKey)
		snap.SecretPresent = true
	}
}

func activeCustomProvider(cfg file, providerID, activeEndpoint string) (*customProvider, bool) {
	for i := range cfg.CustomProviders {
		candidate := &cfg.CustomProviders[i]
		if strings.EqualFold(candidate.Name, providerID) {
			return candidate, false
		}
	}
	identity := endpointIdentity(activeEndpoint)
	if identity != "" {
		var match *customProvider
		for i := range cfg.CustomProviders {
			candidate := &cfg.CustomProviders[i]
			if endpointIdentity(candidate.BaseURL) == identity {
				if match != nil {
					return nil, true
				}
				match = candidate
			}
		}
		return match, false
	}
	return nil, false
}

func endpointIdentity(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ""
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.Path = strings.TrimSuffix(parsed.Path, "/v1")
	return parsed.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (a Adapter) ValidateDesired(fsys fsx.FS, home string, d model.Desired) error {
	if d.SecretRef == "" {
		return nil
	}
	cfg, providerID, activeEndpoint, err := readActiveConfig(fsys, home, d.Provider)
	if err != nil {
		return err
	}
	if p, ambiguous := activeCustomProvider(cfg, providerID, activeEndpoint); ambiguous || p != nil && p.APIKey != "" {
		return exitcode.Errorf(exitcode.Usage, "set secretRef is unsupported while Hermes uses an inline custom_providers API key")
	}
	return nil
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
	if err := a.ValidateDesired(fsys, home, d); err != nil {
		return nil, err
	}
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
			snap, _ := a.Read(fsys, home)
			prov = snap.Provider
		}
		if prov == "" || prov == "auto" {
			prov = "custom"
		}
		prov = strings.TrimPrefix(prov, "custom:")
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
	snap, err := a.Read(fsys, home)
	if err != nil {
		return "", "", err
	}
	cfg, providerID, activeEndpoint, err := readActiveConfig(fsys, home, "")
	if err != nil {
		return "", "", err
	}
	if p, ambiguous := activeCustomProvider(cfg, providerID, activeEndpoint); ambiguous {
		return "", "", exitcode.Errorf(exitcode.Usage, "Hermes custom provider secret is ambiguous")
	} else if p != nil && p.APIKey != "" {
		return "", p.APIKey, nil
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

func (a Adapter) ValidateSecretWrite(fsys fsx.FS, home, provider string) error {
	cfg, providerID, activeEndpoint, err := readActiveConfig(fsys, home, provider)
	if err != nil {
		return err
	}
	if p, ambiguous := activeCustomProvider(cfg, providerID, activeEndpoint); ambiguous || p != nil && p.APIKey != "" {
		return exitcode.Errorf(exitcode.Usage, "secret copy is unsupported while Hermes uses an inline custom_providers API key")
	}
	return nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	if err := a.ValidateSecretWrite(fsys, home, ""); err != nil {
		return err
	}
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

func readActiveConfig(fsys fsx.FS, home, providerOverride string) (file, string, string, error) {
	data, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".hermes", "config.yaml"))
	if err != nil || data == nil {
		return file{}, "", "", err
	}
	var cfg file
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return file{}, "", "", exitcode.Errorf(exitcode.Parse, "Hermes config.yaml is invalid")
	}
	providerID := strings.TrimPrefix(providerOverride, "custom:")
	if providerID == "" {
		var obj modelObj
		raw, _ := yaml.Marshal(cfg.Model)
		_ = yaml.Unmarshal(raw, &obj)
		providerID = strings.TrimPrefix(obj.Provider, "custom:")
	}
	activeEndpoint := ""
	if provider, ok := cfg.Providers[providerID]; ok {
		activeEndpoint = firstNonEmpty(provider.BaseURL, provider.API)
	}
	return cfg, providerID, activeEndpoint, nil
}
