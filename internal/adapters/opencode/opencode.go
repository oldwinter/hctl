package opencode

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/oldwinter/harnessctl/internal/jsonc"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// Adapter reads ~/.config/opencode/opencode.jsonc (or opencode.json).
type Adapter struct{}

func (Adapter) Name() string          { return "opencode" }
func (Adapter) Aliases() []string     { return nil }
func (Adapter) BinaryNames() []string { return []string{"opencode"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{
		filepath.Join(".config", "opencode", "opencode.jsonc"),
		filepath.Join(".config", "opencode", "opencode.json"),
	}
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

func (a Adapter) Read(home string) (model.Snapshot, error) {
	candidates := []string{
		filepath.Join(home, ".config", "opencode", "opencode.jsonc"),
		filepath.Join(home, ".config", "opencode", "opencode.json"),
	}
	snap := model.Snapshot{
		Name:        a.Name(),
		ConfigPaths: candidates,
	}
	var (
		data []byte
		err  error
		used string
	)
	for _, p := range candidates {
		data, err = os.ReadFile(p)
		if err == nil {
			used = p
			break
		}
		if !os.IsNotExist(err) {
			return snap, err
		}
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

func applyNode(snap *model.Snapshot, n providerNode) {
	opts := n.Options
	if opts == nil {
		opts = n.Settings
	}
	if opts == nil {
		return
	}
	if u := asString(opts["baseURL"]); u == "" {
		u = asString(opts["baseUrl"])
		if u == "" {
			u = asString(opts["base_url"])
		}
		if u != "" {
			snap.BaseURLHost = secret.HostOf(u)
		}
	} else {
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
