package claude

import (
	"encoding/json"
	"strings"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
)

// Adapter reads and writes ~/.claude/settings.json.
type Adapter struct{}

func (Adapter) Name() string          { return "claude" }
func (Adapter) Aliases() []string     { return []string{"claude-code"} }
func (Adapter) BinaryNames() []string { return []string{"claude"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".claude/settings.json", ".claude.json"}
}

type file struct {
	Model string            `json:"model"`
	Theme string            `json:"theme,omitempty"`
	Env   map[string]string `json:"env"`
}

func (a Adapter) Read(fsys fsx.FS, home string) (model.Snapshot, error) {
	path := fsys.Join(home, ".claude", "settings.json")
	onboardPath := fsys.Join(home, ".claude.json")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{path, onboardPath}, Provider: "anthropic"}
	data, err := fsys.ReadFile(path)
	if err != nil {
		if !fsx.IsNotExist(err) {
			return snap, err
		}
	} else {
		snap.ConfigFound = true
		var cfg file
		if err := json.Unmarshal(data, &cfg); err != nil {
			snap.ParseError = err.Error()
			noteClaudeOnboarding(fsys, onboardPath, &snap)
			return snap, nil
		}
		snap.DefaultModel = cfg.Model
		if cfg.Theme != "" && cfg.Model == "" && len(cfg.Env) == 0 {
			snap.Notes = append(snap.Notes, "theme wizard leftover — onboarding incomplete")
		}
		if cfg.Env != nil {
			applyClaudeEnv(&snap, cfg.Env)
		}
	}
	noteClaudeOnboarding(fsys, onboardPath, &snap)
	return snap, nil
}

func applyClaudeEnv(snap *model.Snapshot, env map[string]string) {
	if m := env["ANTHROPIC_MODEL"]; m != "" && snap.DefaultModel == "" {
		snap.DefaultModel = m
	}
	if u := env["ANTHROPIC_BASE_URL"]; u != "" {
		snap.BaseURLHost = secret.HostOf(u)
	}
	aliases := map[string]string{}
	for k, v := range env {
		if strings.HasPrefix(k, "ANTHROPIC_DEFAULT_") && strings.HasSuffix(k, "_MODEL") && v != "" {
			aliases[k] = v
		}
	}
	if len(aliases) > 0 {
		snap.Aliases = aliases
	}
	for _, key := range []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"} {
		if val := env[key]; val != "" {
			if name, isRef := secret.EnvRef(val); isRef {
				snap.SecretRef = name
				continue
			}
			snap.SecretFingerprint = secret.Fingerprint(val)
			snap.SecretPresent = true
			break
		}
	}
}

func noteClaudeOnboarding(fsys fsx.FS, path string, snap *model.Snapshot) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		if fsx.IsNotExist(err) {
			snap.Notes = append(snap.Notes, "missing ~/.claude.json — first-run onboarding needed")
		}
		return
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		snap.Notes = append(snap.Notes, "unreadable ~/.claude.json — first-run onboarding needed")
		return
	}
	theme, _ := raw["theme"].(string)
	completed := claudeOnboardingComplete(raw)
	if strings.TrimSpace(theme) == "" || !completed {
		snap.Notes = append(snap.Notes, "incomplete ~/.claude.json (missing theme / onboarding) — first-run onboarding needed")
	}
}

func claudeOnboardingComplete(raw map[string]any) bool {
	for _, key := range []string{"hasCompletedOnboarding", "hasCompletedOnboardingForNewUsers"} {
		switch v := raw[key].(type) {
		case bool:
			if v {
				return true
			}
		case string:
			if strings.EqualFold(v, "true") {
				return true
			}
		}
	}
	return false
}

func (a Adapter) ValidateDesired(fsys fsx.FS, home string, d model.Desired) error {
	if d.Provider != "" {
		return exitcode.Errorf(exitcode.Usage, "set provider is unsupported for claude (provider is implicit anthropic); use set model")
	}
	return nil
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	if err := a.ValidateDesired(fsys, home, d); err != nil {
		return nil, err
	}
	path := fsys.Join(home, ".claude", "settings.json")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return nil, err
	}
	if d.Model != "" {
		data, err = edit.SetJSON(data, []string{"model"}, d.Model)
		if err != nil {
			return nil, err
		}
	}
	if d.SecretRef != "" {
		data, err = edit.SetJSON(data, []string{"env", "ANTHROPIC_AUTH_TOKEN"}, d.SecretRef)
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
	path := fsys.Join(home, ".claude", "settings.json")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil || len(data) == 0 {
		return "", "", err
	}
	var cfg file
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", err
	}
	if cfg.Env == nil {
		return "", "", nil
	}
	for _, key := range []string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"} {
		if val := cfg.Env[key]; val != "" {
			if name, isRef := secret.EnvRef(val); isRef {
				return name, "", nil
			}
			return key, val, nil
		}
	}
	return "", "", nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	path := fsys.Join(home, ".claude", "settings.json")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return err
	}
	if value != "" {
		data, err = edit.SetJSON(data, []string{"env", "ANTHROPIC_AUTH_TOKEN"}, value)
		if err != nil {
			return err
		}
	} else if ref != "" {
		data, err = edit.SetJSON(data, []string{"env", "ANTHROPIC_AUTH_TOKEN"}, ref)
		if err != nil {
			return err
		}
	}
	return fsx.AtomicWrite(fsys, path, data, 0o600)
}
