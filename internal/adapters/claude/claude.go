package claude

import (
	"encoding/json"
	"strings"

	"github.com/oldwinter/harnessctl/internal/edit"
	"github.com/oldwinter/harnessctl/internal/exitcode"
	"github.com/oldwinter/harnessctl/internal/fsx"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads and writes ~/.claude/settings.json.
type Adapter struct{}

func (Adapter) Name() string          { return "claude" }
func (Adapter) Aliases() []string     { return []string{"claude-code"} }
func (Adapter) BinaryNames() []string { return []string{"claude"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".claude/settings.json"}
}

type file struct {
	Model string            `json:"model"`
	Theme string            `json:"theme,omitempty"`
	Env   map[string]string `json:"env"`
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	path := fsys.Join(home, ".claude", "settings.json")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{path}, Provider: "anthropic"}
	data, err := fsys.ReadFile(path)
	if err != nil {
		if fsx.IsNotExist(err) {
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
	if cfg.Theme != "" && cfg.Model == "" && len(cfg.Env) == 0 {
		snap.Notes = append(snap.Notes, "theme wizard leftover — onboarding incomplete")
	}
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

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	if d.Provider != "" && !strings.EqualFold(d.Provider, "anthropic") {
		return nil, exitcode.Errorf(exitcode.Usage, "claude provider is implicit (anthropic); got %q", d.Provider)
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
