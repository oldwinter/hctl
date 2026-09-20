package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/render"
)

func newConfigCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and switch hctl contexts (environments)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get-contexts",
		Short: "List contexts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := opts.loadConfig()
			if err != nil {
				return err
			}
			if opts.wantJSON() {
				return render.JSON(cmd.OutOrStdout(), cfg)
			}
			if err := render.ContextsTable(cmd.OutOrStdout(), cfg); err != nil {
				return err
			}
			if configFileMissing(opts.configPath) {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "Next: config file not created yet; hctl config set-context box --kind ssh --ssh user@host")
				return err
			}
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "current-context",
		Short: "Display the current context name",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := opts.loadConfig()
			if err != nil {
				return err
			}
			if opts.wantJSON() {
				return render.JSON(cmd.OutOrStdout(), map[string]string{
					"currentContext": cfg.CurrentContext,
					"config":         opts.configPath,
				})
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), cfg.CurrentContext); err != nil {
				return err
			}
			if _, err := os.Stat(opts.configPath); os.IsNotExist(err) {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s (not created yet)\n", opts.configPath)
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), opts.configPath)
			return err
		},
	})
	setCtx := &cobra.Command{
		Use:   "set-context NAME",
		Short: "Create or update a context (local or ssh)",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return exitcode.Errorf(exitcode.Usage, "config set-context NAME (example: hctl config set-context box --kind ssh --ssh user@host)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, _ := cmd.Flags().GetString("kind")
			home, _ := cmd.Flags().GetString("home")
			ssh, _ := cmd.Flags().GetString("ssh")
			ident, _ := cmd.Flags().GetString("identity")
			cfg, err := opts.loadConfig()
			if err != nil {
				return err
			}
			nc := config.NamedContext{Name: args[0], Context: config.Context{
				Kind: kind, Home: home, SSH: ssh, IdentityFile: ident,
			}}
			if nc.Context.Kind == "" {
				if ssh != "" {
					nc.Context.Kind = config.KindSSH
				} else {
					nc.Context.Kind = config.KindLocal
				}
			}
			found := false
			for i := range cfg.Contexts {
				if cfg.Contexts[i].Name == args[0] {
					cfg.Contexts[i] = nc
					found = true
					break
				}
			}
			if !found {
				cfg.Contexts = append(cfg.Contexts, nc)
			}
			if err := config.Save(opts.configPath, cfg); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Context %q saved.\n", args[0]); err != nil {
				return err
			}
			if cfg.CurrentContext != args[0] {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "Next: hctl config use-context %s\n", args[0])
			}
			return err
		},
	}
	setCtx.Flags().String("kind", "", "local or ssh")
	setCtx.Flags().String("home", "", "home directory on the target")
	setCtx.Flags().String("ssh", "", "user@host")
	setCtx.Flags().String("identity", "", "SSH IdentityFile")
	cmd.AddCommand(setCtx)
	cmd.AddCommand(&cobra.Command{
		Use:   "use-context NAME",
		Short: "Set the current context (writes the resolved hctl config file)",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return exitcode.Errorf(exitcode.Usage, "config use-context NAME (example: hctl config get-contexts then hctl config use-context box)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := opts.loadConfig()
			if err != nil {
				return err
			}
			if err := cfg.UseContext(args[0]); err != nil {
				hint := fmt.Sprintf("add it under contexts: in %s (see README for a box/ssh example)", opts.configPath)
				if _, statErr := os.Stat(opts.configPath); os.IsNotExist(statErr) {
					hint = fmt.Sprintf("Next: hctl config set-context %s --kind local", args[0])
				}
				return fmt.Errorf("%w; %s", err, hint)
			}
			if err := config.Save(opts.configPath, cfg); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Switched to context %q.\n", args[0])
			return err
		},
	})
	return cmd
}
