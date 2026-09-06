package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oldwinter/harnessctl/internal/config"
	"github.com/oldwinter/harnessctl/internal/render"
)

func newConfigCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and switch harnessctl contexts (environments)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get-contexts",
		Short: "List contexts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := opts.loadConfig()
			if err != nil {
				return writeErr(cmd, err)
			}
			if opts.jsonOut {
				return render.JSON(cmd.OutOrStdout(), cfg)
			}
			return render.ContextsTable(cmd.OutOrStdout(), cfg)
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "current-context",
		Short: "Display the current context name",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := opts.loadConfig()
			if err != nil {
				return writeErr(cmd, err)
			}
			if opts.jsonOut {
				return render.JSON(cmd.OutOrStdout(), map[string]string{"currentContext": cfg.CurrentContext})
			}
			fmt.Fprintln(cmd.OutOrStdout(), cfg.CurrentContext)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "use-context NAME",
		Short: "Set the current context (writes the harnessctl config file)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := opts.loadConfig()
			if err != nil {
				return writeErr(cmd, err)
			}
			if err := cfg.UseContext(args[0]); err != nil {
				return writeErr(cmd, fmt.Errorf("%w; add it under contexts: in %s (see README for a box/ssh example)", err, opts.configPath))
			}
			if err := config.Save(opts.configPath, cfg); err != nil {
				return writeErr(cmd, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Switched to context %q.\n", args[0])
			return nil
		},
	})
	return cmd
}
