package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/harnessctl/internal/adapters"
	"github.com/oldwinter/harnessctl/internal/config"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/render"
)

func newDiffCmd(opts *options) *cobra.Command {
	var (
		aName, bName string
		contexts     string
		homeA, homeB string
	)
	cmd := &cobra.Command{
		Use:   "diff RESOURCE NAME",
		Short: "Compare a harness snapshot across two contexts or two home directories",
		Long: `Compare one harness between two sides.

v0.1 executes locally only. Cross-machine SSH diff is v0.2.
Until then, compare two fixture trees or two home directories:

  harnessctl diff harness codex --home-a testdata/home-a --home-b testdata/home-b
  harnessctl diff harness codex --a mba --b box          # errors if box is ssh
  harnessctl diff harness codex --contexts mba,box
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isHarnessResource(args[0]) {
				return writeErr(cmd, fmt.Errorf("unknown resource %q (want harness)", args[0]))
			}
			ad, err := adapters.ByName(args[1])
			if err != nil {
				return writeErr(cmd, err)
			}
			cfg, err := opts.loadConfig()
			if err != nil {
				return writeErr(cmd, err)
			}
			labelA, labelB, pathA, pathB, err := resolveDiffSides(cfg, aName, bName, contexts, homeA, homeB, opts.home)
			if err != nil {
				return writeErr(cmd, err)
			}
			sa, err := adapters.ReadOne(ad, pathA)
			if err != nil {
				return writeErr(cmd, err)
			}
			sb, err := adapters.ReadOne(ad, pathB)
			if err != nil {
				return writeErr(cmd, err)
			}
			diffs := model.DiffSnapshots(sa, sb)
			if opts.jsonOut {
				return render.JSON(cmd.OutOrStdout(), map[string]any{
					"harness": ad.Name(),
					"a":       labelA,
					"b":       labelB,
					"diff":    diffs,
				})
			}
			return render.Diff(cmd.OutOrStdout(), labelA+" ("+pathA+")", labelB+" ("+pathB+")", diffs)
		},
	}
	cmd.Flags().StringVar(&aName, "a", "", "context A")
	cmd.Flags().StringVar(&bName, "b", "", "context B")
	cmd.Flags().StringVar(&contexts, "contexts", "", "comma-separated pair of contexts (e.g. mba,box)")
	cmd.Flags().StringVar(&homeA, "home-a", "", "home directory A (fixtures / tests; bypasses ssh)")
	cmd.Flags().StringVar(&homeB, "home-b", "", "home directory B (fixtures / tests; bypasses ssh)")
	return cmd
}

func resolveDiffSides(cfg *config.File, aName, bName, contexts, homeA, homeB, homeFlag string) (labelA, labelB, pathA, pathB string, err error) {
	if contexts != "" {
		parts := strings.Split(contexts, ",")
		if len(parts) != 2 {
			return "", "", "", "", fmt.Errorf("--contexts wants exactly two names, got %q", contexts)
		}
		aName, bName = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	if homeA != "" || homeB != "" {
		if homeA == "" || homeB == "" {
			return "", "", "", "", fmt.Errorf("--home-a and --home-b must be used together")
		}
		la, lb := "a", "b"
		if aName != "" {
			la = aName
		}
		if bName != "" {
			lb = bName
		}
		return la, lb, homeA, homeB, nil
	}
	if aName == "" || bName == "" {
		return "", "", "", "", fmt.Errorf("need --a/--b, --contexts NAME,NAME, or --home-a/--home-b")
	}
	_, pathA, err = cfg.ResolveHome(aName, homeFlag)
	if err != nil {
		return "", "", "", "", err
	}
	_, pathB, err = cfg.ResolveHome(bName, homeFlag)
	if err != nil {
		return "", "", "", "", err
	}
	return aName, bName, pathA, pathB, nil
}
