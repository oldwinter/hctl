package codex

import (
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads ~/.codex/config.toml.
type Adapter struct{}

func (Adapter) Name() string          { return "codex" }
func (Adapter) Aliases() []string     { return []string{"openai-codex"} }
func (Adapter) BinaryNames() []string { return []string{"codex"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{filepath.Join(".codex", "config.toml")}
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
	path := filepath.Join(home, ".codex", "config.toml")
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
		if !snap.SecretPresent {
			snap.SecretPresent = false
		}
	}
}
