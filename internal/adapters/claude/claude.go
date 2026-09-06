package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads ~/.claude/settings.json.
type Adapter struct{}

func (Adapter) Name() string          { return "claude" }
func (Adapter) Aliases() []string     { return []string{"claude-code"} }
func (Adapter) BinaryNames() []string { return []string{"claude"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{filepath.Join(".claude", "settings.json")}
}

type file struct {
	Model string            `json:"model"`
	Env   map[string]string `json:"env"`
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	path := filepath.Join(home, ".claude", "settings.json")
	snap := model.Snapshot{
		Name:        a.Name(),
		ConfigPaths: []string{path},
		Provider:    "anthropic",
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
	if err := json.Unmarshal(data, &cfg); err != nil {
		snap.ParseError = err.Error()
		return snap, nil
	}
	snap.DefaultModel = cfg.Model
	if cfg.Env == nil {
		return snap, nil
	}
	if m := cfg.Env["ANTHROPIC_MODEL"]; m != "" && snap.DefaultModel == "" {
		snap.DefaultModel = m
	}
	if u := cfg.Env["ANTHROPIC_BASE_URL"]; u != "" {
		snap.BaseURLHost = secret.HostOf(u)
	}
	aliases := map[string]string{}
	for k, v := range cfg.Env {
		if strings.HasPrefix(k, "ANTHROPIC_DEFAULT_") && strings.HasSuffix(k, "_MODEL") && v != "" {
			aliases[k] = v
		}
	}
	if len(aliases) > 0 {
		snap.Aliases = aliases
	}

	for _, key := range []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"} {
		if val := cfg.Env[key]; val != "" {
			if name, isRef := secret.EnvRef(val); isRef {
				snap.SecretRef = name
				continue
			}
			snap.SecretFingerprint = secret.Fingerprint(val)
			snap.SecretPresent = true
			break
		}
	}
	return snap, nil
}
