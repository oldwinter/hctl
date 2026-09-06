package codex

import (
	"github.com/pelletier/go-toml/v2"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
)

// Adapter reads and writes ~/.codex/config.toml.
type Adapter struct{}

func (Adapter) Name() string          { return "codex" }
func (Adapter) Aliases() []string     { return []string{"openai-codex"} }
func (Adapter) BinaryNames() []string { return []string{"codex"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".codex/config.toml"}
}

type file struct {
	Model                string              `toml:"model"`
	ModelProvider        string              `toml:"model_provider"`
	ModelReasoningEffort string              `toml:"model_reasoning_effort"`
	ModelProviders       map[string]provider `toml:"model_providers"`
	OpenAIBaseURL        string              `toml:"openai_base_url"`
}

type provider struct {
	Name                    string `toml:"name"`
	BaseURL                 string `toml:"base_url"`
	EnvKey                  string `toml:"env_key"`
	ExperimentalBearerToken string `toml:"experimental_bearer_token"`
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	path := fsys.Join(home, ".codex", "config.toml")
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
	snap.DefaultModel = cfg.Model
	snap.Provider = cfg.ModelProvider
	snap.Effort = cfg.ModelReasoningEffort
	if cfg.ModelProvider != "" && cfg.ModelProviders != nil {
		if p, ok := cfg.ModelProviders[cfg.ModelProvider]; ok {
			applyProvider(&snap, p)
		}
	}
	if snap.BaseURLHost == "" && cfg.OpenAIBaseURL != "" {
		snap.BaseURLHost = secret.HostOf(cfg.OpenAIBaseURL)
	}
	if snap.Provider == "" {
		snap.Provider = "openai"
	}
	return snap, nil
}

func applyProvider(snap *model.Snapshot, p provider) {
	if p.BaseURL != "" {
		snap.BaseURLHost = secret.HostOf(p.BaseURL)
	}
	if p.ExperimentalBearerToken != "" {
		snap.SecretFingerprint = secret.Fingerprint(p.ExperimentalBearerToken)
		snap.SecretPresent = true
	}
	if p.EnvKey != "" {
		snap.SecretRef = p.EnvKey
	}
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	path := fsys.Join(home, ".codex", "config.toml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return nil, err
	}
	if d.Model != "" {
		data, err = edit.SetTOML(data, []string{"model"}, d.Model)
		if err != nil {
			return nil, err
		}
	}
	if d.Provider != "" {
		data, err = edit.SetTOML(data, []string{"model_provider"}, d.Provider)
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
		if prov == "" {
			prov = "custom"
		}
		data, err = edit.SetTOML(data, []string{"model_providers", prov, "env_key"}, d.SecretRef)
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
	path := fsys.Join(home, ".codex", "config.toml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil || len(data) == 0 {
		return "", "", err
	}
	var cfg file
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return "", "", err
	}
	if cfg.ModelProviders == nil {
		return "", "", nil
	}
	p := cfg.ModelProviders[cfg.ModelProvider]
	return p.EnvKey, p.ExperimentalBearerToken, nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	path := fsys.Join(home, ".codex", "config.toml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return err
	}
	snap, _ := a.ReadFS(fsys, home)
	prov := snap.Provider
	if prov == "" || prov == "openai" {
		prov = "custom"
	}
	if value != "" {
		data, err = edit.SetTOML(data, []string{"model_providers", prov, "experimental_bearer_token"}, value)
		if err != nil {
			return err
		}
	}
	if ref != "" {
		data, err = edit.SetTOML(data, []string{"model_providers", prov, "env_key"}, ref)
		if err != nil {
			return err
		}
	}
	return fsx.AtomicWrite(fsys, path, data, 0o600)
}
