package codex

import (
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/exitcode"
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

func (a Adapter) Read(fsys fsx.FS, home string) (model.Snapshot, error) {
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

func (a Adapter) ValidateDesired(fsys fsx.FS, home string, d model.Desired) error {
	if d.Provider == "" && d.SecretRef == "" {
		return nil
	}
	cfg, err := readCodexFile(fsys, home)
	if err != nil {
		return err
	}
	if d.SecretRef != "" && d.Provider == "" && cfg.ModelProvider == "" {
		return exitcode.Errorf(exitcode.Usage, "codex secretRef requires a selected model_provider (set provider first)")
	}
	if d.Provider == "" || len(cfg.ModelProviders) == 0 {
		return nil
	}
	if _, ok := cfg.ModelProviders[d.Provider]; ok {
		return nil
	}
	names := make([]string, 0, len(cfg.ModelProviders))
	for name := range cfg.ModelProviders {
		names = append(names, name)
	}
	sort.Strings(names)
	return exitcode.Errorf(exitcode.Usage, "codex provider %q is not in model_providers; have %s", d.Provider, strings.Join(names, ", "))
}

func readCodexFile(fsys fsx.FS, home string) (file, error) {
	data, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".codex", "config.toml"))
	if err != nil || len(data) == 0 {
		return file{}, err
	}
	var cfg file
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return file{}, exitcode.Errorf(exitcode.Parse, "codex config.toml is invalid")
	}
	return cfg, nil
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	if err := a.ValidateDesired(fsys, home, d); err != nil {
		return nil, err
	}
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
			var cfg file
			_ = toml.Unmarshal(data, &cfg)
			prov = cfg.ModelProvider
		}
		if prov == "" {
			return nil, exitcode.Errorf(exitcode.Usage, "codex secretRef requires a selected model_provider (set provider first)")
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

func (a Adapter) ValidateSecretWrite(fsys fsx.FS, home string, d model.Desired) error {
	if d.Provider != "" {
		return nil
	}
	cfg, err := readCodexFile(fsys, home)
	if err != nil {
		return err
	}
	if cfg.ModelProvider == "" {
		return exitcode.Errorf(exitcode.Usage, "codex secret copy requires a selected model_provider (set provider first)")
	}
	return nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	if err := a.ValidateSecretWrite(fsys, home, model.Desired{}); err != nil {
		return err
	}
	path := fsys.Join(home, ".codex", "config.toml")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return err
	}
	var cfg file
	_ = toml.Unmarshal(data, &cfg)
	prov := cfg.ModelProvider
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
