package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/render"
)

func newGetCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:       "get RESOURCE",
		Short:     "List harnesses or default models",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"harnesses", "harness", "models", "model"},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, fsys, home, err := opts.openTarget()
			if err != nil {
				return writeErr(cmd, err)
			}
			snaps, err := adapters.ScanFS(fsys, home)
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
				return writeErr(cmd, fmt.Errorf("unknown resource %q (want harnesses|models)", args[0]))
			}
		},
	}
}
