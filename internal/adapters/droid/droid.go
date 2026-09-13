package droid

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/exitcode"
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
	Model                  string           `json:"model"`
	Provider               string           `json:"provider"`
	BaseURL                string           `json:"baseURL"`
	APIKey                 string           `json:"apiKey"`
	SessionDefaultSettings *sessionDefaults `json:"sessionDefaultSettings"`
	CustomModels           []customModel    `json:"customModels"`
}

type sessionDefaults struct {
	Model string `json:"model"`
}

type customModel struct {
	ID       string `json:"id"`
	Model    string `json:"model"`
	Provider string `json:"provider"`
	BaseURL  string `json:"baseUrl"`
	APIKey   string `json:"apiKey"`
}

func (a Adapter) Read(fsys fsx.FS, home string) (model.Snapshot, error) {
	path := fsys.Join(home, ".factory", "settings.json")
	authPath := fsys.Join(home, ".factory", "auth.json")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{path, authPath}}
	data, err := fsys.ReadFile(path)
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
	if cfg.SessionDefaultSettings != nil && cfg.SessionDefaultSettings.Model != "" {
		snap.DefaultModel = cfg.SessionDefaultSettings.Model
	}
	selected := findCustomModel(cfg.CustomModels, snap.DefaultModel)
	if selected != nil {
		snap.Provider = selected.Provider
		applyEndpointAndSecret(&snap, selected.BaseURL, selected.APIKey)
	} else {
		snap.Provider = cfg.Provider
		applyEndpointAndSecret(&snap, cfg.BaseURL, cfg.APIKey)
	}
	if !snap.SecretPresent && snap.SecretRef == "" && !fsx.Exists(fsys, authPath) {
		snap.Notes = append(snap.Notes, "missing factory auth")
	}
	return snap, nil
}

func findCustomModel(models []customModel, selected string) *customModel {
	for i := range models {
		if models[i].ID == selected {
			return &models[i]
		}
	}
	return nil
}

func applyEndpointAndSecret(snap *model.Snapshot, baseURL, apiKey string) {
	snap.BaseURLHost = secret.HostOf(baseURL)
	if apiKey == "" {
		return
	}
	if name, isRef := secret.EnvRef(apiKey); isRef {
		snap.SecretRef = name
	} else {
		snap.SecretFingerprint = secret.Fingerprint(apiKey)
		snap.SecretPresent = true
	}
}

func (Adapter) UnsupportedDesiredFields() []string {
	return []string{"provider", "secret-ref"}
}

func (a Adapter) ValidateDesired(fsys fsx.FS, home string, d model.Desired) error {
	cfg, err := readSettings(fsys, home)
	if err != nil {
		return err
	}
	if cfg.SessionDefaultSettings == nil {
		return nil
	}
	if d.Provider != "" {
		return exitcode.Errorf(exitcode.Usage, "set provider is unsupported for current Factory Droid custom models; select a model id")
	}
	if d.SecretRef != "" {
		return exitcode.Errorf(exitcode.Usage, "set secretRef is unsupported for current Factory Droid custom models")
	}
	if d.Model != "" && len(cfg.CustomModels) > 0 && findCustomModel(cfg.CustomModels, d.Model) == nil {
		names := make([]string, 0, len(cfg.CustomModels))
		for _, m := range cfg.CustomModels {
			if m.ID != "" {
				names = append(names, m.ID)
			}
		}
		sort.Strings(names)
		return exitcode.Errorf(exitcode.Usage, "droid model %q is not in customModels; have %s", d.Model, strings.Join(names, ", "))
	}
	return nil
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	if err := a.ValidateDesired(fsys, home, d); err != nil {
		return nil, err
	}
	path := fsys.Join(home, ".factory", "settings.json")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return nil, err
	}
	if d.Model != "" {
		var cfg settings
		_ = json.Unmarshal(data, &cfg)
		keyPath := []string{"model"}
		if cfg.SessionDefaultSettings != nil {
			keyPath = []string{"sessionDefaultSettings", "model"}
		}
		data, err = edit.SetJSON(data, keyPath, d.Model)
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
	value = cfg.APIKey
	if cfg.SessionDefaultSettings != nil {
		if selected := findCustomModel(cfg.CustomModels, cfg.SessionDefaultSettings.Model); selected != nil {
			value = selected.APIKey
		}
	}
	if name, isRef := secret.EnvRef(value); isRef {
		return name, "", nil
	}
	return "", value, nil
}

func (a Adapter) ValidateSecretWrite(fsys fsx.FS, home, provider string) error {
	cfg, err := readSettings(fsys, home)
	if err != nil {
		return err
	}
	if cfg.SessionDefaultSettings != nil {
		return exitcode.Errorf(exitcode.Usage, "secret copy is unsupported for current Factory Droid custom models")
	}
	return nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	if err := a.ValidateSecretWrite(fsys, home, ""); err != nil {
		return err
	}
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

func readSettings(fsys fsx.FS, home string) (settings, error) {
	data, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".factory", "settings.json"))
	if err != nil || data == nil {
		return settings{}, err
	}
	var cfg settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return settings{}, exitcode.Errorf(exitcode.Parse, "Factory Droid settings.json is invalid")
	}
	return cfg, nil
}
