package droid

import (
	"encoding/json"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
)

// Adapter reads ~/.factory/settings.json (Factory Droid).
type Adapter struct{}

func (Adapter) Name() string          { return "droid" }
func (Adapter) Aliases() []string     { return []string{"factory"} }
func (Adapter) BinaryNames() []string { return []string{"droid"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".factory/settings.json", ".factory/auth.json"}
}

type settings struct {
	Model    string `json:"model"`
	Provider string `json:"provider"`
	BaseURL  string `json:"baseURL"`
	APIKey   string `json:"apiKey"`
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	path := fsys.Join(home, ".factory", "settings.json")
	authPath := fsys.Join(home, ".factory", "auth.json")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{path, authPath}}
	data, err := fsys.ReadFile(path)
	if err != nil {
		if fsx.IsNotExist(err) {
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
	snap.Provider = cfg.Provider
	if cfg.BaseURL != "" {
		snap.BaseURLHost = secret.HostOf(cfg.BaseURL)
	}
	if cfg.APIKey != "" {
		if name, isRef := secret.EnvRef(cfg.APIKey); isRef {
			snap.SecretRef = name
		} else {
			snap.SecretFingerprint = secret.Fingerprint(cfg.APIKey)
			snap.SecretPresent = true
		}
	}
	if !snap.SecretPresent && snap.SecretRef == "" && !fsx.Exists(fsys, authPath) {
		snap.Notes = append(snap.Notes, "missing factory auth")
	}
	return snap, nil
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	path := fsys.Join(home, ".factory", "settings.json")
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
	data, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".factory", "settings.json"))
	if err != nil || len(data) == 0 {
		return "", "", err
	}
	var cfg settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", err
	}
	if name, isRef := secret.EnvRef(cfg.APIKey); isRef {
		return name, "", nil
	}
	return "", cfg.APIKey, nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	path := fsys.Join(home, ".factory", "settings.json")
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
