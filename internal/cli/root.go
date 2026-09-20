package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/ownership"
)

// Version is the CLI version string. Override via -ldflags.
var (
	Version = "1.0.1"
	Commit  = "unknown"
	Date    = "unknown"
)

type options struct {
	jsonOut           bool
	output            string
	home              string
	configPath        string
	contextName       string
	allowManaged      bool
	ownershipManifest string
	noProbe           bool
}

func (o *options) resolveHomeEnv() {
	if o.home == "" {
		o.home = config.Getenv("HOME")
	}
	paths := config.ResolvePaths(o.configPath)
	o.configPath = paths.Config
}

func (o *options) loadConfig() (*config.File, error) {
	return config.Load(o.configPath)
}

func (o *options) activeContextName(cfg *config.File) string {
	if o.contextName != "" {
		return o.contextName
	}
	return cfg.CurrentContext
}

func (o *options) wantJSON() bool {
	return o.jsonOut || strings.EqualFold(strings.TrimSpace(o.output), "json")
}

func (o *options) wantWide() bool {
	return !o.wantJSON() && strings.EqualFold(strings.TrimSpace(o.output), "wide")
}

func (o *options) validateOutput() error {
	switch strings.ToLower(strings.TrimSpace(o.output)) {
	case "", "json", "wide":
		return nil
	default:
		return exitcode.Errorf(exitcode.Usage, "unknown output %q (want json|wide). Next: hctl get harnesses -o json", o.output)
	}
}

// NewRoot builds the kubectl-style command tree.
func NewRoot() *cobra.Command {
	opts := &options{}
	use := "hctl"
	if filepath.Base(os.Args[0]) == "harnessctl" {
		use = "harnessctl"
	}
	root := &cobra.Command{
		Use:   use,
		Short: "kubectl-style control plane for AI coding-agent harnesses",
		Long: `hctl (also installed as harnessctl) inventories coding-agent harness configs
(Codex, Claude Code, Grok Build, Hermes, OpenCode, …) across machines —
without dispatching agents.

Read, set, apply, and later sync harness configs. Secrets are never printed.

Context = environment / machine (mba, box), not a Kubernetes cluster.`,
		Example: `  hctl get harnesses
  hctl get harnesses -o json
  hctl get harness codex
  hctl doctor
  hctl describe harness codex`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			opts.resolveHomeEnv()
			return opts.validateOutput()
		},
	}
	root.PersistentFlags().BoolVar(&opts.jsonOut, "json", false, "emit JSON instead of a table")
	root.PersistentFlags().StringVarP(&opts.output, "output", "o", "", "output format: json|wide (wide adds config paths)")
	root.PersistentFlags().StringVar(&opts.home, "home", "", "override user home used to locate harness configs (also HCTL_HOME / HARNESSCTL_HOME)")
	root.PersistentFlags().StringVar(&opts.configPath, "config", "", "path to hctl kubeconfig-like file (default ~/.hctl/config.yaml; also HCTL_CONFIG / HARNESSCTL_CONFIG)")
	root.PersistentFlags().StringVar(&opts.contextName, "context", "", "context to use for this command (overrides current-context)")
	root.PersistentFlags().BoolVar(&opts.allowManaged, "allow-managed", false, "allow a temporary write to dotfiles-managed harness destinations")
	root.PersistentFlags().StringVar(&opts.ownershipManifest, "ownership-manifest", "", "target manifest path (absolute or ~/; overrides ownership pointer discovery)")
	root.PersistentFlags().BoolVar(&opts.noProbe, "no-probe", false, "inspect config without version or login subprocess probes")

	root.AddCommand(newVersionCmd(opts))
	root.AddCommand(newConfigCmd(opts))
	root.AddCommand(newGetCmd(opts))
	root.AddCommand(newDescribeCmd(opts))
	root.AddCommand(newDoctorCmd(opts))
	root.AddCommand(newDiffCmd(opts))
	root.AddCommand(newSetCmd(opts))
	root.AddCommand(newApplyCmd(opts))
	root.AddCommand(newSyncCmd(opts))
	root.AddCommand(newCompletionCmd())
	return root
}

func (o *options) ownershipOptions() ownership.Options {
	return ownership.Options{Manifest: o.ownershipManifest, AllowManaged: o.allowManaged}
}

func configFileMissing(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func writeErr(cmd *cobra.Command, err error) error {
	return err
}

// Execute runs the CLI and annotates cobra unknown-command errors with a next step.
func Execute() error {
	return Run(NewRoot())
}

// Run executes cmd and rewrites unknown-command errors (SilenceErrors hides cobra's hint).
func Run(cmd *cobra.Command) error {
	return annotateUnknownCommand(cmd.Execute())
}

func annotateUnknownCommand(err error) error {
	if err == nil {
		return nil
	}
	var coded *exitcode.Error
	if errors.As(err, &coded) {
		return err
	}
	if !strings.Contains(err.Error(), `unknown command "`) {
		return err
	}
	return exitcode.Errorf(exitcode.Usage, "%s\ntry: hctl --help\nhctl --home testdata/home-a --config testdata/harnessctl.yaml get harnesses", err)
}
