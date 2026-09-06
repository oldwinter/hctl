package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/harnessctl/internal/adapters"
	"github.com/oldwinter/harnessctl/internal/config"
	"github.com/oldwinter/harnessctl/internal/desired"
	"github.com/oldwinter/harnessctl/internal/exitcode"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/render"
)

func newDiffCmd(opts *options) *cobra.Command {
	var (
		aName, bName string
		contexts     string
		homeA, homeB string
		filename     string
	)
	cmd := &cobra.Command{
		Use:   "diff [RESOURCE NAME]",
		Short: "Compare harness snapshots or a desired-state file",
		Long: `Compare one harness between two sides, or desired state vs current:

  harnessctl diff harness codex --home-a testdata/home-a --home-b testdata/home-b
  harnessctl diff harness codex --contexts mba,box
  harnessctl diff -f testdata/desired.toml
`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename != "" {
				return runDiffDesired(cmd, opts, filename)
			}
			if len(args) != 2 {
				return exitcode.Errorf(exitcode.Usage, "diff harness NAME (or diff -f FILE)")
			}
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
	cmd.Flags().StringVarP(&filename, "filename", "f", "", "desired-state file to compare against current context")
	return cmd
}

func runDiffDesired(cmd *cobra.Command, opts *options, filename string) error {
	want, err := desired.Load(filename)
	if err != nil {
		return err
	}
	_, fsys, home, err := opts.openTarget()
	if err != nil {
		return err
	}
	snaps, err := adapters.ScanFS(fsys, home)
	if err != nil {
		return err
	}
	changes := desired.DiffAgainst(want, snaps)
	if opts.jsonOut {
		return render.JSON(cmd.OutOrStdout(), map[string]any{
			"file":    filename,
			"changes": changes,
		})
	}
	rep := model.ApplyReport{DryRun: true, Changes: changes}
	return render.ApplyReport(cmd.OutOrStdout(), rep)
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
