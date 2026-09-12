package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/oldwinter/hctl/internal/fsx"
)

const (
	APIVersion = "harnessctl/v1"
	KindConfig = "Config"
	KindLocal  = "local"
	KindSSH    = "ssh"

	SourceFlag      = "flag"
	SourceEnv       = "env"
	SourceCanonical = "canonical"
	SourceLegacy    = "legacy"
)

// File is the kubeconfig-like document stored at ~/.hctl/config.yaml
// (or an existing ~/.harnessctl/config.yaml).
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

// Paths is the resolved config identity. Writes go to Config in place.
type Paths struct {
	Config string
	Source string
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

// Getenv reads HCTL_<name>, then HARNESSCTL_<name>. name is HOME, CONFIG, BACKUP_DIR, or SSH.
func Getenv(name string) string {
	if v := strings.TrimSpace(os.Getenv("HCTL_" + name)); v != "" {
		return v
	}
	return strings.TrimSpace(os.Getenv("HARNESSCTL_" + name))
}

// ResolvePaths picks one config file. Order: --config, HCTL_CONFIG,
// HARNESSCTL_CONFIG, existing ~/.hctl/config.yaml, existing
// ~/.harnessctl/config.yaml, else create ~/.hctl/config.yaml.
// It never copies or deletes ~/.harnessctl.
func ResolvePaths(flagPath string) Paths {
	if p := strings.TrimSpace(flagPath); p != "" {
		return Paths{Config: p, Source: SourceFlag}
	}
	if p := Getenv("CONFIG"); p != "" {
		return Paths{Config: p, Source: SourceEnv}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{Config: filepath.Join(".hctl", "config.yaml"), Source: SourceCanonical}
	}
	canonical := filepath.Join(home, ".hctl", "config.yaml")
	legacy := filepath.Join(home, ".harnessctl", "config.yaml")
	if fsx.Exists(fsx.Local{}, canonical) {
		return Paths{Config: canonical, Source: SourceCanonical}
	}
	if fsx.Exists(fsx.Local{}, legacy) {
		return Paths{Config: legacy, Source: SourceLegacy}
	}
	return Paths{Config: canonical, Source: SourceCanonical}
}

// DefaultPath is the resolved config path when no --config flag is set.
func DefaultPath() string {
	return ResolvePaths("").Config
}

// BackupDir is $HCTL_BACKUP_DIR, else $HARNESSCTL_BACKUP_DIR, else backups/
// beside the resolved config file.
func BackupDir(configPath string) string {
	if d := Getenv("BACKUP_DIR"); d != "" {
		return d
	}
	if configPath == "" {
		configPath = DefaultPath()
	}
	return filepath.Join(filepath.Dir(configPath), "backups")
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

// Save writes f with 0600 perms: backup (if the file exists and changed),
// temp file, rename, re-read verify. Identical bytes are a no-op.
func Save(path string, f *File) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && bytes.Equal(existing, data) {
		return nil
	}
	if err == nil && len(existing) > 0 {
		if _, bakErr := fsx.BackupLocal(BackupDir(path), "config", path, existing); bakErr != nil {
			return bakErr
		}
	}
	if err := fsx.AtomicWrite(fsx.Local{}, path, data, 0o600); err != nil {
		return err
	}
	got, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("verify %s: %w", path, err)
	}
	if !bytes.Equal(got, data) {
		return fmt.Errorf("verify %s: re-read mismatch", path)
	}
	return nil
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
