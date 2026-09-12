package opencode

import (
	"strings"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/jsonc"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
)

// Adapter reads and writes ~/.config/opencode/opencode.jsonc (or .json).
type Adapter struct{}

func (Adapter) Name() string          { return "opencode" }
func (Adapter) Aliases() []string     { return nil }
func (Adapter) BinaryNames() []string { return []string{"opencode"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".config/opencode/opencode.jsonc", ".config/opencode/opencode.json"}
}

type file struct {
	Model     string                  `json:"model"`
	Provider  map[string]providerNode `json:"provider"`
	Providers map[string]providerNode `json:"providers"`
}

type providerNode struct {
	Options  map[string]any `json:"options"`
	Settings map[string]any `json:"settings"`
}

func (a Adapter) Read(fsys fsx.FS, home string) (model.Snapshot, error) {
	used, data, err := readConfig(fsys, home)
	snap := model.Snapshot{
		Name:        a.Name(),
		ConfigPaths: []string{fsys.Join(home, ".config", "opencode", "opencode.jsonc"), fsys.Join(home, ".config", "opencode", "opencode.json")},
	}
	if err != nil {
		return snap, err
	}
	if used == "" {
		return snap, nil
	}
	snap.ConfigFound = true
	snap.ConfigPaths = []string{used}
	var cfg file
	if err := jsonc.Unmarshal(data, &cfg); err != nil {
		snap.ParseError = err.Error()
		return snap, nil
	}
	snap.DefaultModel = cfg.Model
	provID := ""
	if i := strings.IndexByte(cfg.Model, '/'); i > 0 {
		provID = cfg.Model[:i]
		snap.Provider = provID
	}
	nodes := cfg.Provider
	if nodes == nil {
		nodes = cfg.Providers
	}
	if nodes != nil {
		if provID != "" {
			if n, ok := nodes[provID]; ok {
				applyNode(&snap, n)
			}
		} else if len(nodes) == 1 {
			for name, n := range nodes {
				snap.Provider = name
				applyNode(&snap, n)
			}
		}
	}
	return snap, nil
}

func readConfig(fsys fsx.FS, home string) (used string, data []byte, err error) {
	for _, rel := range [][]string{{".config", "opencode", "opencode.jsonc"}, {".config", "opencode", "opencode.json"}} {
		p := fsys.Join(append([]string{home}, rel...)...)
		data, err = fsys.ReadFile(p)
		if err == nil {
			return p, data, nil
		}
		if !fsx.IsNotExist(err) {
			return "", nil, err
		}
	}
	return "", nil, nil
}

func applyNode(snap *model.Snapshot, n providerNode) {
	opts := n.Options
	if opts == nil {
		opts = n.Settings
	}
	if opts == nil {
		return
	}
	u := asString(opts["baseURL"])
	if u == "" {
		u = asString(opts["baseUrl"])
	}
	if u == "" {
		u = asString(opts["base_url"])
	}
	if u != "" {
		snap.BaseURLHost = secret.HostOf(u)
	}
	if e := asString(opts["reasoningEffort"]); e != "" {
		snap.Effort = e
	}
	key := asString(opts["apiKey"])
	if key == "" {
		key = asString(opts["api_key"])
	}
	if key == "" {
		return
	}
	if name, isRef := secret.EnvRef(key); isRef {
		snap.SecretRef = name
		return
	}
	snap.SecretFingerprint = secret.Fingerprint(key)
	snap.SecretPresent = true
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	used, data, err := readConfig(fsys, home)
	if err != nil {
		return nil, err
	}
	if used == "" {
		used = fsys.Join(home, ".config", "opencode", "opencode.jsonc")
	}
	modelVal := d.Model
	if d.Provider != "" {
		if modelVal == "" {
			snap, _ := a.Read(fsys, home)
			modelVal = snap.DefaultModel
		}
		if i := strings.IndexByte(modelVal, '/'); i > 0 {
			modelVal = d.Provider + modelVal[i:]
		} else if modelVal != "" {
			modelVal = d.Provider + "/" + modelVal
		} else {
			modelVal = d.Provider + "/default"
		}
	}
	if modelVal != "" {
		data, err = edit.SetJSONC(data, []string{"model"}, modelVal)
		if err != nil {
			return nil, err
		}
	}
	if d.SecretRef != "" {
		prov := d.Provider
		if prov == "" {
			if i := strings.IndexByte(modelVal, '/'); i > 0 {
				prov = modelVal[:i]
			}
		}
		if prov == "" {
			prov = "custom"
		}
		data, err = edit.SetJSONC(data, []string{"provider", prov, "options", "apiKey"}, "{env:"+d.SecretRef+"}")
		if err != nil {
			return nil, err
		}
	}
	if err := fsx.AtomicWrite(fsys, used, data, 0o600); err != nil {
		return nil, err
	}
	return []string{used}, nil
}

func (a Adapter) PeekSecret(fsys fsx.FS, home string) (ref, value string, err error) {
	_, data, err := readConfig(fsys, home)
	if err != nil || len(data) == 0 {
		return "", "", err
	}
	var cfg file
	if err := jsonc.Unmarshal(data, &cfg); err != nil {
		return "", "", err
	}
	nodes := cfg.Provider
	if nodes == nil {
		nodes = cfg.Providers
	}
	prov := ""
	if i := strings.IndexByte(cfg.Model, '/'); i > 0 {
		prov = cfg.Model[:i]
	}
	n, ok := nodes[prov]
	if !ok {
		return "", "", nil
	}
	opts := n.Options
	if opts == nil {
		opts = n.Settings
	}
	if opts == nil {
		return "", "", nil
	}
	key := asString(opts["apiKey"])
	if key == "" {
		key = asString(opts["api_key"])
	}
	if name, isRef := secret.EnvRef(key); isRef {
		return name, "", nil
	}
	return "", key, nil
}

func (a Adapter) WriteSecret(fsys fsx.FS, home, ref, value string) error {
	used, data, err := readConfig(fsys, home)
	if err != nil {
		return err
	}
	if used == "" {
		used = fsys.Join(home, ".config", "opencode", "opencode.jsonc")
	}
	snap, _ := a.Read(fsys, home)
	prov := snap.Provider
	if prov == "" {
		prov = "custom"
	}
	val := value
	if val == "" && ref != "" {
		val = "{env:" + ref + "}"
	}
	data, err = edit.SetJSONC(data, []string{"provider", prov, "options", "apiKey"}, val)
	if err != nil {
		return err
	}
	return fsx.AtomicWrite(fsys, used, data, 0o600)
}
