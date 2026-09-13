package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/render"
)

const getResources = "harnesses|harness|models|model"

func newGetCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:   "get RESOURCE [NAME]",
		Short: "List harnesses or default models",
		Long: `List harness inventory or default models.

Valid resources: harnesses (alias harness), models (alias model).
NAME selects one harness (official name or alias), like kubectl get.

Examples:
  hctl get harnesses
  hctl get harnesses -o json
  hctl get harnesses -o wide
  hctl get harness codex
  hctl get models`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return exitcode.Errorf(exitcode.Usage, "missing resource; want %s. Next: hctl get harnesses", getResources)
			}
			if len(args) > 2 {
				return exitcode.Errorf(exitcode.Usage, "too many args; want %s get RESOURCE [NAME] (example: hctl get harness codex)", cmd.Root().Name())
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
			name := ""
			if len(args) == 2 {
				name = args[1]
			}
			snaps, err = pickSnapshots(snaps, name)
			if err != nil {
				return writeErr(cmd, err)
			}
			res := strings.ToLower(args[0])
			switch res {
			case "harnesses", "harness":
				if opts.wantJSON() {
					if name != "" {
						return render.JSON(cmd.OutOrStdout(), snaps[0])
					}
					return render.JSON(cmd.OutOrStdout(), snaps)
				}
				if err := render.HarnessesTable(cmd.OutOrStdout(), snaps, opts.wantWide()); err != nil {
					return err
				}
				if name == "" && noneConfigured(snaps) {
					_, err := fmt.Fprintln(cmd.ErrOrStderr(), "Next: hctl get harnesses -o wide")
					return err
				}
				return nil
			case "models", "model":
				if opts.wantJSON() {
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
					if name != "" {
						return render.JSON(cmd.OutOrStdout(), rows[0])
					}
					return render.JSON(cmd.OutOrStdout(), rows)
				}
				if err := render.ModelsTable(cmd.OutOrStdout(), snaps); err != nil {
					return err
				}
			default:
				return writeErr(cmd, exitcode.Errorf(exitcode.Usage, "unknown resource %q (want %s). Next: hctl get harnesses", args[0], getResources))
			}
			return nil
		},
	}
}

func pickSnapshots(snaps []model.Snapshot, name string) ([]model.Snapshot, error) {
	if strings.TrimSpace(name) == "" {
		return snaps, nil
	}
	ad, err := adapters.ByName(name)
	if err != nil {
		return nil, err
	}
	for _, s := range snaps {
		if s.Name == ad.Name() {
			return []model.Snapshot{s}, nil
		}
	}
	return nil, fmt.Errorf("harness %q not in inventory", name)
}

func noneConfigured(snaps []model.Snapshot) bool {
	if len(snaps) == 0 {
		return false
	}
	for _, s := range snaps {
		if s.ConfigFound {
			return false
		}
	}
	return true
}
