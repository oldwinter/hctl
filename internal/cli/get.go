package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/render"
)

const getResources = "harnesses|harness|models|model"

func newGetCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "get RESOURCE",
		Short: "List harnesses or default models",
		Long: `List harness inventory or default models.

Valid resources: harnesses (alias harness), models (alias model).

Examples:
  hctl get harnesses
  hctl get models
  hctl get harnesses -o wide`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return exitcode.Errorf(exitcode.Usage, "missing resource; want %s", getResources)
			}
			if len(args) > 1 {
				return exitcode.Errorf(exitcode.Usage, "too many args; want %s get RESOURCE", cmd.Root().Name())
			}
			return nil
		},
		Aliases:   []string{"list"},
		ValidArgs: []string{"harnesses", "harness", "models", "model"},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, fsys, home, err := opts.openTarget()
			if err != nil {
				return writeErr(cmd, err)
			}
			snaps, err := adapters.Scan(fsys, home)
			if err != nil {
				return writeErr(cmd, err)
			}
			res := strings.ToLower(args[0])
			switch res {
			case "harnesses", "harness":
				if opts.jsonOut || opts.output == "json" {
					return render.JSON(cmd.OutOrStdout(), snaps)
				}
				return render.HarnessesTable(cmd.OutOrStdout(), snaps, opts.output == "wide")
			case "models", "model":
				if opts.jsonOut {
					type row struct {
						Harness  string `json:"harness"`
						Provider string `json:"provider,omitempty"`
						Model    string `json:"model,omitempty"`
						Effort   string `json:"effort,omitempty"`
						Host     string `json:"host,omitempty"`
					}
					rows := make([]row, 0, len(snaps))
					for _, s := range snaps {
						rows = append(rows, row{
							Harness:  s.Name,
							Provider: s.Provider,
							Model:    s.DefaultModel,
							Effort:   s.Effort,
							Host:     s.BaseURLHost,
						})
					}
					return render.JSON(cmd.OutOrStdout(), rows)
				}
				return render.ModelsTable(cmd.OutOrStdout(), snaps)
			default:
				return writeErr(cmd, exitcode.Errorf(exitcode.Usage, "unknown resource %q (want %s)", args[0], getResources))
			}
		},
	}
}
