package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/model"
	"github.com/oldwinter/hctl/internal/secret"
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
func HarnessesTable(w io.Writer, snaps []model.Snapshot, wide bool) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	header := "NAME\tINSTALLED\tVERSION\tPROVIDER\tMODEL\tHOST\tSECRET"
	if wide {
		header += "\tCONFIG"
	}
	if _, err := fmt.Fprintln(tw, header); err != nil {
		return err
	}
	for _, s := range snaps {
		var err error
		if wide {
			_, err = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				s.Name, yesNo(s.Installed), dash(s.Version), dash(s.Provider),
				dash(s.DefaultModel), dash(s.BaseURLHost), secretCell(s), strings.Join(s.ConfigPaths, ","))
		} else {
			_, err = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				s.Name,
				yesNo(s.Installed),
				dash(s.Version),
				dash(s.Provider),
				dash(s.DefaultModel),
				dash(s.BaseURLHost),
				secretCell(s),
			)
		}
		if err != nil {
			return err
		}
	}
	return tw.Flush()
}

// ModelsTable prints the per-harness default model summary.
func ModelsTable(w io.Writer, snaps []model.Snapshot) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "HARNESS\tPROVIDER\tMODEL\tEFFORT\tHOST"); err != nil {
		return err
	}
	for _, s := range snaps {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			s.Name, dash(s.Provider), dash(s.DefaultModel), dash(s.Effort), dash(s.BaseURLHost)); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// Describe writes a kubectl-like block for one snapshot.
func Describe(w io.Writer, contextName string, s model.Snapshot) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Name:              %s\n", s.Name)
	if len(s.NameAliases) > 0 {
		fmt.Fprintf(&b, "Also known as:     %s\n", strings.Join(s.NameAliases, ", "))
	}
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
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NAME\tINSTALLED\tCONFIG\tKEY\tONBOARDING\tDRIFT\tMESSAGE"); err != nil {
		return err
	}
	for _, c := range checks {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			c.Name, c.Installed, c.Config, c.Key, c.Onboarding, dash(c.Drift), c.Message); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := io.WriteString(w, secret.Redact(b.String()))
	return err
}

// ContextsTable prints kubeconfig-like contexts.
func ContextsTable(w io.Writer, f *config.File) error {
	if f == nil || len(f.Contexts) == 0 {
		_, err := fmt.Fprintln(w, "No contexts configured.")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, "Next: hctl config set-context box --kind ssh --ssh user@host")
		return err
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "CURRENT\tNAME\tKIND\tHOME\tSSH"); err != nil {
		return err
	}
	for _, c := range f.Contexts {
		cur := ""
		if c.Name == f.CurrentContext {
			cur = "*"
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			cur, c.Name, dash(c.Context.Kind), dash(c.Context.Home), dash(c.Context.SSH)); err != nil {
			return err
		}
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
	if _, err := fmt.Fprintln(tw, "FIELD\tA\tB"); err != nil {
		return err
	}
	for _, d := range diffs {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\n", d.Field, empty(d.A), empty(d.B)); err != nil {
			return err
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := io.WriteString(w, secret.Redact(b.String()))
	return err
}

func secretCell(s model.Snapshot) string {
	var parts []string
	if s.SecretFingerprint != "" {
		parts = append(parts, "sha256:"+s.SecretFingerprint)
	}
	if s.SecretRef != "" {
		parts = append(parts, "env:"+s.SecretRef)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	if s.SecretPresent {
		return "present"
	}
	return "-"
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

func hasSecretAction(secrets []model.SecretCopy) bool {
	for _, s := range secrets {
		if s.Action != "" && s.Action != "none" {
			return true
		}
	}
	return false
}

func secretCopied(secrets []model.SecretCopy) bool {
	for _, s := range secrets {
		if s.Copied {
			return true
		}
	}
	return false
}

func jsoncWriteWarning(r model.ApplyReport) bool {
	for _, c := range r.Changes {
		if strings.HasSuffix(strings.ToLower(c.Path), ".jsonc") {
			return true
		}
	}
	return false
}

// ApplyReport prints a set/apply/sync summary.
func ApplyReport(w io.Writer, r model.ApplyReport) error {
	var b strings.Builder
	if r.DryRun {
		fmt.Fprintln(&b, "dry-run: no files written")
	} else if r.Verified || secretCopied(r.Secrets) {
		fmt.Fprintln(&b, "verified: ok")
	}
	if len(r.Changes) == 0 && !hasSecretAction(r.Secrets) {
		fmt.Fprintln(&b, "no changes")
	} else if len(r.Changes) > 0 {
		tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(tw, "HARNESS\tFIELD\tFROM\tTO\tPATH"); err != nil {
			return err
		}
		for _, c := range r.Changes {
			if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", c.Harness, c.Field, empty(c.From), empty(c.To), dash(c.Path)); err != nil {
				return err
			}
		}
		if err := tw.Flush(); err != nil {
			return err
		}
	}
	for _, c := range r.Secrets {
		if c.Action == "" || c.Action == "none" {
			continue
		}
		line := "secret " + c.Harness + " action=" + c.Action
		if c.From != "" {
			line += " from=sha256:" + c.From
		}
		if c.To != "" {
			line += " to=sha256:" + c.To
		}
		if c.Ref != "" {
			line += " ref=" + c.Ref
		}
		fmt.Fprintln(&b, line)
	}
	if jsoncWriteWarning(r) {
		fmt.Fprintln(&b, "note: writing JSONC drops comments and trailing commas")
	}
	for _, n := range r.Notes {
		fmt.Fprintf(&b, "note: %s\n", n)
	}
	for _, bak := range r.Backups {
		fmt.Fprintf(&b, "backup: %s\n", bak)
	}
	_, err := io.WriteString(w, secret.Redact(b.String()))
	return err
}
