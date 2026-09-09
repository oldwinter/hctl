package pi

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
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
	DefaultModel    string `json:"defaultModel"`
	DefaultProvider string `json:"defaultProvider"`
	Model           string `json:"model"`
	Provider        string `json:"provider"`
	BaseURL         string `json:"baseURL"`
	BaseURL2        string `json:"base_url"`
}

type modelsFile struct {
	Providers map[string]provider `json:"providers"`
}

type provider struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
}

type authCredential struct {
	Type    string            `json:"type"`
	Key     *string           `json:"key"`
	Env     map[string]string `json:"env"`
	Access  *string           `json:"access"`
	Refresh *string           `json:"refresh"`
	Expires *float64          `json:"expires"`
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	setPath := fsys.Join(home, ".pi", "agent", "settings.json")
	modPath := fsys.Join(home, ".pi", "agent", "models.json")
	authPath := fsys.Join(home, ".pi", "agent", "auth.json")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{setPath, modPath, authPath}}

	setData, err := fsx.ReadMaybe(fsys, setPath)
	if err != nil {
		return snap, err
	}
	modData, err := fsx.ReadMaybe(fsys, modPath)
	if err != nil {
		return snap, err
	}
	authData, err := fsx.ReadMaybe(fsys, authPath)
	if err != nil {
		return snap, err
	}
	snap.ConfigFound = setData != nil || modData != nil || authData != nil

	var cfg settings
	if setData != nil {
		if err := json.Unmarshal(setData, &cfg); err != nil {
			snap.ParseError = "settings.json: invalid JSON"
			return snap, nil
		}
	}
	snap.DefaultModel = first(cfg.DefaultModel, cfg.Model)
	snap.Provider = first(cfg.DefaultProvider, cfg.Provider)

	var models modelsFile
	if modData != nil {
		if err := json.Unmarshal(modData, &models); err != nil || models.Providers == nil {
			snap.ParseError = "models.json: invalid provider configuration"
			return snap, nil
		}
	}
	selectedModelProvider, hasModelProvider := models.Providers[snap.Provider]
	if hasModelProvider {
		snap.BaseURLHost = secret.HostOf(selectedModelProvider.BaseURL)
	}
	if snap.BaseURLHost == "" {
		snap.BaseURLHost = secret.HostOf(first(cfg.BaseURL, cfg.BaseURL2))
	}

	credentials, err := parseAuth(authData)
	if err != nil {
		snap.ParseError = "auth.json: invalid provider credential"
		return snap, nil
	}
	if credential, ok := credentials[snap.Provider]; ok {
		applyCredential(&snap, credential)
		return snap, nil
	}
	if hasModelProvider {
		applyConfigSecret(&snap, selectedModelProvider.APIKey)
	}
	return snap, nil
}

func (a Adapter) ValidateDesired(fsys fsx.FS, home string, d model.Desired) error {
	if d.SecretRef == "" {
		return nil
	}
	if !envName.MatchString(d.SecretRef) {
		return exitcode.Errorf(exitcode.Usage, "pi secretRef must be an environment variable name")
	}
	provider, err := selectedProvider(fsys, home, d.Provider)
	if err != nil {
		return err
	}
	if provider == "" {
		return exitcode.Errorf(exitcode.Usage, "pi secretRef requires a selected provider")
	}
	return validateCredentialWrite(fsys, home, provider)
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	if err := a.ValidateDesired(fsys, home, d); err != nil {
		return nil, err
	}
	setPath := fsys.Join(home, ".pi", "agent", "settings.json")
	authPath := fsys.Join(home, ".pi", "agent", "auth.json")
	setData, err := fsx.ReadMaybe(fsys, setPath)
	if err != nil {
		return nil, err
	}
	if d.Model != "" {
		setData, err = edit.SetJSON(setData, []string{"defaultModel"}, d.Model)
		if err != nil {
			return nil, err
		}
	}
	if d.Provider != "" {
		setData, err = edit.SetJSON(setData, []string{"defaultProvider"}, d.Provider)
		if err != nil {
			return nil, err
		}
	}

	var authData []byte
	if d.SecretRef != "" {
		provider, err := selectedProvider(fsys, home, d.Provider)
		if err != nil {
			return nil, err
		}
		authData, err = updatedAuth(fsys, authPath, provider, "$"+d.SecretRef)
		if err != nil {
			return nil, err
		}
	}

	paths := make([]string, 0, 2)
	if d.SecretRef != "" {
		if err := fsx.AtomicWrite(fsys, authPath, authData, 0o600); err != nil {
			return nil, err
		}
		paths = append(paths, authPath)
	}
	if d.Model != "" || d.Provider != "" {
		if err := fsx.AtomicWrite(fsys, setPath, setData, 0o600); err != nil {
			return paths, err
		}
		paths = append(paths, setPath)
	}
	return paths, nil
}

func (a Adapter) PeekSecret(fsys fsx.FS, home string) (ref, value string, err error) {
	providerID, err := selectedProvider(fsys, home, "")
	if err != nil || providerID == "" {
		return "", "", err
	}
	authPath := fsys.Join(home, ".pi", "agent", "auth.json")
	authData, err := fsx.ReadMaybe(fsys, authPath)
	if err != nil {
		return "", "", err
	}
	credentials, err := parseAuth(authData)
	if err != nil {
		return "", "", exitcode.Errorf(exitcode.Parse, "pi auth.json has an invalid provider credential")
	}
	if credential, ok := credentials[providerID]; ok {
		switch credential.Type {
		case "api_key":
			if credential.Key != nil && *credential.Key != "" {
				return credentialSecret(credential)
			}
			return "", "", nil
		case "oauth":
			return "", "", exitcode.Errorf(exitcode.Usage, "pi OAuth credential copy is unsupported; use pi login on the destination")
		}
	}

	models, err := readModels(fsys, home)
	if err != nil {
		return "", "", err
	}
	return configSecret(models.Providers[providerID].APIKey)
}

func (a Adapter) ValidateSecretWrite(fsys fsx.FS, home, provider string) error {
	providerID, err := selectedProvider(fsys, home, provider)
	if err != nil {
		return err
	}
	if providerID == "" {
		return exitcode.Errorf(exitcode.Usage, "pi secret copy requires a selected provider")
	}
	return validateCredentialWrite(fsys, home, providerID)
}

func validateCredentialWrite(fsys fsx.FS, home, providerID string) error {
	data, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".pi", "agent", "auth.json"))
	if err != nil {
		return err
	}
	credentials, err := parseAuth(data)
	if err != nil {
		return exitcode.Errorf(exitcode.Parse, "pi auth.json has an invalid provider credential")
	}
	if credential, ok := credentials[providerID]; ok && credential.Type == "oauth" {
		return exitcode.Errorf(exitcode.Usage, "pi destination provider uses OAuth; credential overwrite is unsupported")
	}
	return nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	if err := a.ValidateSecretWrite(fsys, home, ""); err != nil {
		return err
	}
	providerID, err := selectedProvider(fsys, home, "")
	if err != nil {
		return err
	}
	configured := value
	if configured == "" && ref != "" {
		if !envName.MatchString(ref) {
			return exitcode.Errorf(exitcode.Usage, "pi secretRef must be an environment variable name")
		}
		configured = "$" + ref
	}
	if configured == "" {
		return nil
	}
	path := fsys.Join(home, ".pi", "agent", "auth.json")
	data, err := updatedAuth(fsys, path, providerID, configured)
	if err != nil {
		return err
	}
	return fsx.AtomicWrite(fsys, path, data, 0o600)
}

func selectedProvider(fsys fsx.FS, home, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	data, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".pi", "agent", "settings.json"))
	if err != nil || data == nil {
		return "", err
	}
	var cfg settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", exitcode.Errorf(exitcode.Parse, "pi settings.json is invalid")
	}
	return first(cfg.DefaultProvider, cfg.Provider), nil
}

func readModels(fsys fsx.FS, home string) (modelsFile, error) {
	data, err := fsx.ReadMaybe(fsys, fsys.Join(home, ".pi", "agent", "models.json"))
	if err != nil || data == nil {
		return modelsFile{Providers: map[string]provider{}}, err
	}
	var models modelsFile
	if err := json.Unmarshal(data, &models); err != nil || models.Providers == nil {
		return modelsFile{}, exitcode.Errorf(exitcode.Parse, "pi models.json has invalid provider configuration")
	}
	return models, nil
}

func parseAuth(data []byte) (map[string]authCredential, error) {
	if data == nil {
		return map[string]authCredential{}, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil || raw == nil {
		return nil, fmt.Errorf("invalid auth object")
	}
	out := make(map[string]authCredential, len(raw))
	for providerID, value := range raw {
		var credential authCredential
		if err := json.Unmarshal(value, &credential); err != nil {
			return nil, fmt.Errorf("invalid credential for provider")
		}
		switch credential.Type {
		case "api_key":
		case "oauth":
			if credential.Access == nil || credential.Refresh == nil || credential.Expires == nil {
				return nil, fmt.Errorf("invalid OAuth credential")
			}
		default:
			return nil, fmt.Errorf("invalid credential type")
		}
		out[providerID] = credential
	}
	return out, nil
}

func updatedAuth(fsys fsx.FS, path, providerID, configured string) ([]byte, error) {
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if data != nil {
		if err := json.Unmarshal(data, &root); err != nil || root == nil {
			return nil, exitcode.Errorf(exitcode.Parse, "pi auth.json is invalid")
		}
	} else {
		root = map[string]any{}
	}
	credential, _ := root[providerID].(map[string]any)
	if credential == nil {
		credential = map[string]any{}
	}
	delete(credential, "access")
	delete(credential, "refresh")
	delete(credential, "expires")
	credential["type"] = "api_key"
	credential["key"] = configured
	root[providerID] = credential
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func applyCredential(snap *model.Snapshot, credential authCredential) {
	switch credential.Type {
	case "api_key":
		if credential.Key == nil || *credential.Key == "" {
			return
		}
		ref, raw, err := credentialSecret(credential)
		applyObservedSecret(snap, ref, raw, err)
	case "oauth":
		if credential.Access != nil && *credential.Access != "" {
			snap.SecretFingerprint = secret.Fingerprint(*credential.Access)
			snap.SecretPresent = true
			snap.Notes = append(snap.Notes, "selected provider uses OAuth")
		}
	}
}

func applyConfigSecret(snap *model.Snapshot, value string) {
	ref, raw, err := configSecret(value)
	applyObservedSecret(snap, ref, raw, err)
}

func applyObservedSecret(snap *model.Snapshot, ref, raw string, err error) {
	if err != nil {
		snap.Notes = append(snap.Notes, "selected provider uses an unresolved API key expression")
		return
	}
	if ref != "" {
		snap.SecretRef = ref
	}
	if raw != "" {
		snap.SecretFingerprint = secret.Fingerprint(raw)
		snap.SecretPresent = true
	}
}

func credentialSecret(credential authCredential) (ref, raw string, err error) {
	if credential.Key == nil {
		return "", "", nil
	}
	ref, raw, err = configSecret(*credential.Key)
	if err == nil && ref != "" && credential.Env[ref] != "" {
		raw = credential.Env[ref]
	}
	return ref, raw, err
}

func configSecret(value string) (ref, raw string, err error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", nil
	}
	if strings.HasPrefix(value, "!") {
		return "", "", exitcode.Errorf(exitcode.Usage, "pi command-based API key copy is unsupported")
	}
	if match := bracedEnv.FindStringSubmatch(value); match != nil {
		return match[1], "", nil
	}
	if match := simpleEnv.FindStringSubmatch(value); match != nil {
		return match[1], "", nil
	}
	if strings.Contains(value, "$") {
		return "", "", exitcode.Errorf(exitcode.Usage, "pi interpolated API key copy is unsupported")
	}
	return "", value, nil
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

var (
	envName   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	simpleEnv = regexp.MustCompile(`^\$([A-Za-z_][A-Za-z0-9_]*)$`)
	bracedEnv = regexp.MustCompile(`^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$`)
)
