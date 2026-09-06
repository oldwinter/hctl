package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// userHomeDir is overridden in tests.
var userHomeDir = os.UserHomeDir
var yamlMarshal = yaml.Marshal
var saveOverride func(string, *File) error

// SetSaveOverride is for tests.
func SetSaveOverride(fn func(string, *File) error) { saveOverride = fn }

const (
	APIVersion = "harnessctl/v1"
	KindConfig = "Config"
	KindLocal  = "local"
	KindSSH    = "ssh"
)

// ErrSSHNotImplemented is returned when a command tries to execute against an
// SSH context. Cross-machine inventory lands in v0.2.
var ErrSSHNotImplemented = errors.New("ssh contexts are not implemented yet (planned for v0.2); use a local context, --home, or --home-a/--home-b to compare fixture trees")

// File is the kubeconfig-like document stored at ~/.harnessctl/config.yaml.
type File struct {
	APIVersion     string         `yaml:"apiVersion" json:"apiVersion"`
	Kind           string         `yaml:"kind" json:"kind"`
	CurrentContext string         `yaml:"current-context" json:"currentContext"`
	Contexts       []NamedContext `yaml:"contexts" json:"contexts"`
}

// NamedContext pairs a name with its target.
type NamedContext struct {
	Name    string  `yaml:"name" json:"name"`
	Context Context `yaml:"context" json:"context"`
}

// Context is an environment / machine target (not a kube cluster).
type Context struct {
	Kind         string `yaml:"kind" json:"kind"`                     // local | ssh
	Home         string `yaml:"home,omitempty" json:"home,omitempty"` // optional home override
	SSH          string `yaml:"ssh,omitempty" json:"ssh,omitempty"`   // user@host
	User         string `yaml:"user,omitempty" json:"user,omitempty"`
	Host         string `yaml:"host,omitempty" json:"host,omitempty"`
	IdentityFile string `yaml:"identityFile,omitempty" json:"identityFile,omitempty"`
}

// Target returns user@host for SSH contexts.
func (c Context) Target() string {
	if strings.TrimSpace(c.SSH) != "" {
		return c.SSH
	}
	if c.User != "" && c.Host != "" {
		return c.User + "@" + c.Host
	}
	return c.Host
}

// Default returns the built-in mba (local) context. box is documented, not seeded.
func Default() *File {
	return &File{
		APIVersion:     APIVersion,
		Kind:           KindConfig,
		CurrentContext: "mba",
		Contexts: []NamedContext{
			{Name: "mba", Context: Context{Kind: KindLocal}},
		},
	}
}

// DefaultPath is ~/.harnessctl/config.yaml (or $HARNESSCTL_CONFIG if set).
func DefaultPath() string {
	if p := strings.TrimSpace(os.Getenv("HARNESSCTL_CONFIG")); p != "" {
		return p
	}
	home, err := userHomeDir()
	if err != nil {
		return filepath.Join(".harnessctl", "config.yaml")
	}
	return filepath.Join(home, ".harnessctl", "config.yaml")
}

// Load reads path. A missing file returns Default() with no error.
func Load(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if f.APIVersion == "" {
		f.APIVersion = APIVersion
	}
	if f.Kind == "" {
		f.Kind = KindConfig
	}
	if len(f.Contexts) == 0 {
		return Default(), nil
	}
	if f.CurrentContext == "" {
		f.CurrentContext = f.Contexts[0].Name
	}
	for i := range f.Contexts {
		if f.Contexts[i].Context.Kind == "" {
			if f.Contexts[i].Context.SSH != "" {
				f.Contexts[i].Context.Kind = KindSSH
			} else {
				f.Contexts[i].Context.Kind = KindLocal
			}
		}
	}
	return &f, nil
}

// Save writes f with 0600 perms, creating the parent directory as 0700.
func Save(path string, f *File) error {
	if saveOverride != nil {
		return saveOverride(path, f)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yamlMarshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Get returns the named context.
func (f *File) Get(name string) (NamedContext, error) {
	for _, c := range f.Contexts {
		if c.Name == name {
			return c, nil
		}
	}
	return NamedContext{}, fmt.Errorf("context %q not found", name)
}

// UseContext sets current-context. The context must already exist.
func (f *File) UseContext(name string) error {
	if _, err := f.Get(name); err != nil {
		return err
	}
	f.CurrentContext = name
	return nil
}

// Current returns the current named context.
func (f *File) Current() (NamedContext, error) {
	if f.CurrentContext == "" {
		return NamedContext{}, fmt.Errorf("current-context is empty")
	}
	return f.Get(f.CurrentContext)
}

// ResolveHome returns the local home directory to scan for the context.
// homeFlag / HARNESSCTL_HOME always wins and allows fixture tests even for ssh.
func (f *File) ResolveHome(name, homeFlag string) (NamedContext, string, error) {
	nc, err := f.Get(name)
	if err != nil {
		return NamedContext{}, "", err
	}
	if homeFlag != "" {
		return nc, homeFlag, nil
	}
	if nc.Context.Kind == KindSSH && homeFlag == "" {
		if nc.Context.Home != "" {
			return nc, nc.Context.Home, nil
		}
		return nc, "", fmt.Errorf("context %q: %w", name, ErrSSHNotImplemented)
	}
	if nc.Context.Home != "" {
		return nc, expandHome(nc.Context.Home), nil
	}
	home, err := userHomeDir()
	if err != nil {
		return nc, "", err
	}
	return nc, home, nil
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := userHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
