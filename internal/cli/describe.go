package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/oldwinter/hctl/internal/adapters"
	"github.com/oldwinter/hctl/internal/exitcode"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/render"
)

func newDescribeCmd(opts *options) *cobra.Command {
	return &cobra.Command{
		Use:     "describe RESOURCE NAME",
		Short:   "Show details of a harness snapshot",
		Example: `  hctl describe harness codex`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return exitcode.Errorf(exitcode.Usage, "describe harness NAME (example: hctl describe harness codex)")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if !isHarnessResource(args[0]) {
				return exitcode.Errorf(exitcode.Usage, "unknown resource %q (want harness)", args[0])
			}
			ctxName, fsys, home, err := opts.openTarget()
			if err != nil {
				return err
			}
			a, err := adapters.ByName(args[1])
			if err != nil {
				return err
			}
			snap, err := adapters.ReadOne(a, fsys, home)
			if err != nil {
				return err
			}
			appendProviderWriteNote(&snap, a)
			if opts.wantJSON() {
				return render.JSON(cmd.OutOrStdout(), snap)
			}
			return render.Describe(cmd.OutOrStdout(), ctxName, snap)
		},
	}
}

func appendProviderWriteNote(snap *model.Snapshot, a adapters.Adapter) {
	u, ok := a.(interface{ UnsupportedDesiredFields() []string })
	if !ok {
		return
	}
	for _, n := range u.UnsupportedDesiredFields() {
		if n != "provider" {
			continue
		}
		switch a.Name() {
		case "claude":
			snap.Notes = append(snap.Notes, "provider is implicit anthropic and is not writable")
		case "grok":
			snap.Notes = append(snap.Notes, "provider is inferred from base_url and is not writable")
		case "droid":
			snap.Notes = append(snap.Notes, "provider is not writable for current Factory Droid custom models")
		}
	}
}

func isHarnessResource(s string) bool {
	s = strings.ToLower(s)
	return s == "harness" || s == "harnesses"
}
