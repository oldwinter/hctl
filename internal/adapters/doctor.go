package adapters

import (
	"strings"

	"github.com/oldwinter/harnessctl/internal/model"
)

// Doctor derives check rows from snapshots.
func Doctor(snaps []model.Snapshot) []model.DoctorCheck {
	out := make([]model.DoctorCheck, 0, len(snaps))
	for _, s := range snaps {
		c := model.DoctorCheck{Name: s.Name}
		if s.Installed {
			c.Installed = "ok"
		} else {
			c.Installed = "missing"
		}
		switch {
		case s.ParseError != "":
			c.Config = "error"
		case s.ConfigFound:
			c.Config = "ok"
		default:
			c.Config = "missing"
		}
		switch {
		case s.SecretFingerprint != "" || s.SecretPresent:
			c.Key = "ok"
		case s.SecretRef != "":
			c.Key = "env-ref"
		default:
			c.Key = "missing"
		}

		needsOnboarding := false
		var reasons []string
		if !s.ConfigFound {
			needsOnboarding = true
			reasons = append(reasons, "no config file")
		}
		if s.DefaultModel == "" && s.ConfigFound {
			needsOnboarding = true
			reasons = append(reasons, "no default model")
		}
		if s.ParseError != "" {
			needsOnboarding = true
			reasons = append(reasons, "parse error")
		}
		for _, n := range s.Notes {
			if strings.Contains(strings.ToLower(n), "onboard") || strings.Contains(strings.ToLower(n), "not implemented") {
				if strings.Contains(strings.ToLower(n), "not implemented") && !s.ConfigFound && !s.Installed {
					// stub with nothing on disk is just "not present"
					continue
				}
			}
		}
		if needsOnboarding {
			c.Onboarding = "needed"
			c.Message = strings.Join(reasons, "; ")
			if s.ParseError != "" {
				c.Message = s.ParseError
			}
		} else {
			c.Onboarding = "ok"
		}
		if len(s.Notes) > 0 && c.Message == "" {
			c.Message = strings.Join(s.Notes, "; ")
		}
		out = append(out, c)
	}
	return out
}
