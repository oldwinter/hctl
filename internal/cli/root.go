package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/config"
)

// Version is the CLI version string. Override via -ldflags.
var (
	Version = "1.0.1"
	Commit  = "unknown"
	Date    = "unknown"
)

type options struct {
	jsonOut     bool
	output      string
	home        string
	configPath  string
	contextName string
}

func (o *options) resolveHomeEnv() {
	if o.home == "" {
		o.home = os.Getenv("HARNESSCTL_HOME")
	}
	if o.configPath == "" {
		o.configPath = os.Getenv("HARNESSCTL_CONFIG")
	}
	if o.configPath == "" {
		o.configPath = config.DefaultPath()
	}
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

func (o *options) scanHome(cfg *config.File) (contextName, home string, err error) {
	name := o.activeContextName(cfg)
	_, home, err = cfg.ResolveHome(name, o.home)
	return name, home, err
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
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			opts.resolveHomeEnv()
		},
	}
	root.PersistentFlags().BoolVar(&opts.jsonOut, "json", false, "emit JSON instead of a table")
	root.PersistentFlags().StringVarP(&opts.output, "output", "o", "", "output format: json|wide (wide adds config paths)")
	root.PersistentFlags().StringVar(&opts.home, "home", "", "override user home used to locate harness configs (also HARNESSCTL_HOME)")
	root.PersistentFlags().StringVar(&opts.configPath, "config", "", "path to harnessctl kubeconfig-like file (also HARNESSCTL_CONFIG)")
	root.PersistentFlags().StringVar(&opts.contextName, "context", "", "context to use for this command (overrides current-context)")

	root.AddCommand(newVersionCmd())
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

func writeErr(cmd *cobra.Command, err error) error {
	return err
}
