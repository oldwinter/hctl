package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/desired"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/render"
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

  hctl diff harness codex --home-a testdata/home-a --home-b testdata/home-b
  hctl diff harness codex --contexts mba,box
  hctl diff -f testdata/desired.toml
`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if filename != "" {
				return runDiffDesired(cmd, opts, filename)
			}
			if len(args) != 2 {
				return exitcode.Errorf(exitcode.Usage, "diff harness NAME (or diff -f FILE) (example: hctl diff harness codex --home-a testdata/home-a --home-b testdata/home-b)")
			}
			if !isHarnessResource(args[0]) {
				return writeErr(cmd, exitcode.Errorf(exitcode.Usage, "unknown resource %q (want harness)", args[0]))
			}
			ad, err := adapters.ByName(args[1])
			if err != nil {
				return writeErr(cmd, err)
			}
			cfg, err := opts.loadConfig()
			if err != nil {
				return writeErr(cmd, err)
			}
			labelA, labelB, fsA, homeA2, fsB, homeB2, err := resolveDiffFS(opts, cfg, aName, bName, contexts, homeA, homeB)
			if err != nil {
				return writeErr(cmd, err)
			}
			sa, err := adapters.ReadOne(ad, fsA, homeA2)
			if err != nil {
				return writeErr(cmd, err)
			}
			sb, err := adapters.ReadOne(ad, fsB, homeB2)
			if err != nil {
				return writeErr(cmd, err)
			}
			diffs := model.DiffSnapshots(sa, sb)
			if opts.wantJSON() {
				return render.JSON(cmd.OutOrStdout(), map[string]any{
					"harness": ad.Name(),
					"a":       labelA,
					"b":       labelB,
					"diff":    diffs,
				})
			}
			return render.Diff(cmd.OutOrStdout(), labelA+" ("+homeA2+")", labelB+" ("+homeB2+")", diffs)
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
	for name := range want.Harnesses {
		if _, err := adapters.ByName(name); err != nil {
			return err
		}
	}
	_, fsys, home, err := opts.openTarget()
	if err != nil {
		return err
	}
	snaps, err := adapters.Scan(fsys, home)
	if err != nil {
		return err
	}
	changes := desired.DiffAgainst(want, snaps)
	if opts.wantJSON() {
		return render.JSON(cmd.OutOrStdout(), map[string]any{
			"file":    filename,
			"changes": changes,
		})
	}
	rep := model.ApplyReport{DryRun: true, Changes: changes}
	return render.ApplyReport(cmd.OutOrStdout(), rep)
}

func resolveDiffFS(opts *options, cfg *config.File, aName, bName, contexts, homeA, homeB string) (labelA, labelB string, fsA fsx.FS, pathA string, fsB fsx.FS, pathB string, err error) {
	if contexts != "" {
		parts := strings.Split(contexts, ",")
		if len(parts) != 2 {
			return "", "", nil, "", nil, "", fmt.Errorf("--contexts wants exactly two names, got %q", contexts)
		}
		aName, bName = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	if homeA != "" || homeB != "" {
		if homeA == "" || homeB == "" {
			return "", "", nil, "", nil, "", fmt.Errorf("--home-a and --home-b must be used together")
		}
		la, lb := "a", "b"
		if aName != "" {
			la = aName
		}
		if bName != "" {
			lb = bName
		}
		return la, lb, fsx.Local{}, homeA, fsx.Local{}, homeB, nil
	}
	if aName == "" || bName == "" {
		return "", "", nil, "", nil, "", exitcode.Errorf(exitcode.Usage, "diff harness NAME needs --home-a/--home-b, --contexts mba,box, or --a/--b (example: hctl diff harness codex --home-a testdata/home-a --home-b testdata/home-b)")
	}
	_, fsA, pathA, err = opts.openNamed(cfg, aName)
	if err != nil {
		return "", "", nil, "", nil, "", err
	}
	_, fsB, pathB, err = opts.openNamed(cfg, bName)
	if err != nil {
		return "", "", nil, "", nil, "", err
	}
	return aName, bName, fsA, pathA, fsB, pathB, nil
}
