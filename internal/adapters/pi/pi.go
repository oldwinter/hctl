package pi

import (
	"encoding/json"

	"github.com/oldwinter/harnessctl/internal/edit"
	"github.com/oldwinter/harnessctl/internal/fsx"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads ~/.pi/agent/{settings,models,auth}.json.
type Adapter struct{}

func (Adapter) Name() string          { return "pi" }
func (Adapter) Aliases() []string     { return nil }
func (Adapter) BinaryNames() []string { return []string{"pi"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".pi/agent/settings.json", ".pi/agent/models.json", ".pi/agent/auth.json"}
}

type settings struct {
	Model        string `json:"model"`
	DefaultModel string `json:"defaultModel"`
	Provider     string `json:"provider"`
	BaseURL      string `json:"baseURL"`
	BaseURL2     string `json:"base_url"`
}

type auth struct {
	APIKey      string `json:"apiKey"`
	AccessToken string `json:"accessToken"`
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	setPath := fsys.Join(home, ".pi", "agent", "settings.json")
	authPath := fsys.Join(home, ".pi", "agent", "auth.json")
	modPath := fsys.Join(home, ".pi", "agent", "models.json")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{setPath, modPath, authPath}}
	data, err := fsys.ReadFile(setPath)
	if err != nil {
		if fsx.IsNotExist(err) {
			if fsx.Exists(fsys, authPath) {
				snap.ConfigFound = true
			}
			return snap, nil
		}
		return snap, err
	}
	snap.ConfigFound = true
	var cfg settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		snap.ParseError = err.Error()
		return snap, nil
	}
	snap.DefaultModel = cfg.Model
	if snap.DefaultModel == "" {
		snap.DefaultModel = cfg.DefaultModel
	}
	snap.Provider = cfg.Provider
	if u := cfg.BaseURL; u == "" {
		u = cfg.BaseURL2
		if u != "" {
			snap.BaseURLHost = secret.HostOf(u)
		}
	} else {
		snap.BaseURLHost = secret.HostOf(u)
	}
	if raw, err := fsx.ReadMaybe(fsys, authPath); err == nil && len(raw) > 0 {
		var afile auth
		if err := json.Unmarshal(raw, &afile); err == nil {
			val := afile.APIKey
			if val == "" {
				val = afile.AccessToken
			}
			if val != "" {
				if name, isRef := secret.EnvRef(val); isRef {
					snap.SecretRef = name
				} else {
					snap.SecretFingerprint = secret.Fingerprint(val)
					snap.SecretPresent = true
				}
			}
		}
	}
	return snap, nil
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	path := fsys.Join(home, ".pi", "agent", "settings.json")
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
	if d.Provider != "" {
		data, err = edit.SetJSON(data, []string{"provider"}, d.Provider)
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
	raw, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".pi", "agent", "auth.json"))
	if err != nil || len(raw) == 0 {
		return "", "", err
	}
	var afile auth
	if err := json.Unmarshal(raw, &afile); err != nil {
		return "", "", err
	}
	val := afile.APIKey
	if val == "" {
		val = afile.AccessToken
	}
	if name, isRef := secret.EnvRef(val); isRef {
		return name, "", nil
	}
	return "", val, nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	path := fsys.Join(home, ".pi", "agent", "auth.json")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return err
	}
	val := value
	if val == "" {
		val = ref
	}
	data, err = edit.SetJSON(data, []string{"apiKey"}, val)
	if err != nil {
		return err
	}
	return fsx.AtomicWrite(fsys, path, data, 0o600)
}
