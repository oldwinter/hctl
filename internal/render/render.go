package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/oldwinter/harnessctl/internal/config"
	"github.com/oldwinter/harnessctl/internal/model"
	"github.com/oldwinter/harnessctl/internal/secret"
)

// JSON writes indented JSON. A final redact pass refuses to emit raw keys.
func JSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, secret.Redact(string(data))+"\n")
	return err
}

// HarnessesTable prints the inventory table.
func HarnessesTable(w io.Writer, snaps []model.Snapshot) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tINSTALLED\tVERSION\tPROVIDER\tMODEL\tHOST\tSECRET")
	for _, s := range snaps {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			s.Name,
			yesNo(s.Installed),
			dash(s.Version),
			dash(s.Provider),
			dash(s.DefaultModel),
			dash(s.BaseURLHost),
			secretCell(s),
		)
	}
	return tw.Flush()
}

// ModelsTable prints the per-harness default model summary.
func ModelsTable(w io.Writer, snaps []model.Snapshot) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "HARNESS\tPROVIDER\tMODEL\tEFFORT\tHOST")
	for _, s := range snaps {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			s.Name, dash(s.Provider), dash(s.DefaultModel), dash(s.Effort), dash(s.BaseURLHost))
	}
	return tw.Flush()
}

// Describe writes a kubectl-like block for one snapshot.
func Describe(w io.Writer, contextName string, s model.Snapshot) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Name:              %s\n", s.Name)
	fmt.Fprintf(&b, "Context:           %s\n", contextName)
	fmt.Fprintf(&b, "Installed:         %s\n", installedLine(s))
	fmt.Fprintf(&b, "Config Found:      %t\n", s.ConfigFound)
	fmt.Fprintf(&b, "Config Paths:      %s\n", strings.Join(s.ConfigPaths, ", "))
	fmt.Fprintf(&b, "Provider:          %s\n", dash(s.Provider))
	fmt.Fprintf(&b, "Default Model:     %s\n", dash(s.DefaultModel))
	fmt.Fprintf(&b, "Effort:            %s\n", dash(s.Effort))
	fmt.Fprintf(&b, "Base Host:         %s\n", dash(s.BaseURLHost))
	fmt.Fprintf(&b, "Secret:            %s\n", secretCell(s))
	if len(s.Aliases) > 0 {
		fmt.Fprintf(&b, "Aliases:\n")
		for k, v := range s.Aliases {
			fmt.Fprintf(&b, "  %s: %s\n", k, v)
		}
	}
	if s.ParseError != "" {
		fmt.Fprintf(&b, "Parse Error:       %s\n", s.ParseError)
	}
	for _, n := range s.Notes {
		fmt.Fprintf(&b, "Note:              %s\n", n)
	}
	_, err := io.WriteString(w, secret.Redact(b.String()))
	return err
}

// DoctorTable prints doctor rows.
func DoctorTable(w io.Writer, checks []model.DoctorCheck) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tINSTALLED\tCONFIG\tKEY\tONBOARDING\tMESSAGE")
	for _, c := range checks {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			c.Name, c.Installed, c.Config, c.Key, c.Onboarding, c.Message)
	}
	return tw.Flush()
}

// ContextsTable prints kubeconfig-like contexts.
func ContextsTable(w io.Writer, f *config.File) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "CURRENT\tNAME\tKIND\tHOME\tSSH")
	for _, c := range f.Contexts {
		cur := ""
		if c.Name == f.CurrentContext {
			cur = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			cur, c.Name, dash(c.Context.Kind), dash(c.Context.Home), dash(c.Context.SSH))
	}
	return tw.Flush()
}

// Diff writes a field-level comparison. Labels are context/home names.
func Diff(w io.Writer, labelA, labelB string, diffs []model.FieldDiff) error {
	if len(diffs) == 0 {
		_, err := fmt.Fprintf(w, "no differences\n")
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "--- %s\n+++ %s\n\n", labelA, labelB)
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "FIELD\tA\tB")
	for _, d := range diffs {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", d.Field, empty(d.A), empty(d.B))
	}
	_ = tw.Flush()
	_, err := io.WriteString(w, secret.Redact(b.String()))
	return err
}

func secretCell(s model.Snapshot) string {
	switch {
	case s.SecretFingerprint != "":
		return "sha256:" + s.SecretFingerprint
	case s.SecretRef != "":
		return "env:" + s.SecretRef
	case s.SecretPresent:
		return "present"
	default:
		return "-"
	}
}

func installedLine(s model.Snapshot) string {
	if !s.Installed {
		return "no"
	}
	if s.Version != "" {
		return s.InstalledPath + " (" + s.Version + ")"
	}
	return s.InstalledPath
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func dash(v string) string {
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func empty(v string) string {
	if v == "" {
		return "-"
	}
	return v
}

// ApplyReport prints a set/apply/sync summary.
func ApplyReport(w io.Writer, r model.ApplyReport) error {
	var b strings.Builder
	if r.DryRun {
		fmt.Fprintln(&b, "dry-run: no files written")
	} else if r.Verified {
		fmt.Fprintln(&b, "verified: ok")
	}
	if len(r.Changes) == 0 {
		fmt.Fprintln(&b, "no changes")
	} else {
		tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "HARNESS\tFIELD\tFROM\tTO\tPATH")
		for _, c := range r.Changes {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", c.Harness, c.Field, empty(c.From), empty(c.To), dash(c.Path))
		}
		_ = tw.Flush()
	}
	for _, bak := range r.Backups {
		fmt.Fprintf(&b, "backup: %s\n", bak)
	}
	_, err := io.WriteString(w, secret.Redact(b.String()))
	return err
}
