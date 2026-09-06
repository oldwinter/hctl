package adapters

import (
	"fmt"
	"strings"

	"github.com/oldwinter/harnessctl/internal/model"
)

// Doctor derives check rows from snapshots, including key-drift by shared host.
func Doctor(snaps []model.Snapshot) []model.DoctorCheck {
	driftHosts := keyDriftHosts(snaps)
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
			low := strings.ToLower(n)
			if strings.Contains(low, "onboard") || strings.Contains(low, "wizard") || strings.Contains(low, "login not") || (strings.Contains(low, "missing") && strings.Contains(low, "auth")) {
				needsOnboarding = true
				reasons = append(reasons, n)
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
		if s.BaseURLHost != "" && driftHosts[s.BaseURLHost] {
			c.Drift = "key-drift"
			msg := fmt.Sprintf("key drift on host %s", s.BaseURLHost)
			if c.Message == "" {
				c.Message = msg
			} else {
				c.Message = c.Message + "; " + msg
			}
		}
		if len(s.Notes) > 0 && c.Message == "" {
			c.Message = strings.Join(s.Notes, "; ")
		}
		out = append(out, c)
	}
	return out
}

func keyDriftHosts(snaps []model.Snapshot) map[string]bool {
	type set map[string]struct{}
	byHost := map[string]set{}
	for _, s := range snaps {
		if s.BaseURLHost == "" || s.SecretFingerprint == "" {
			continue
		}
		if byHost[s.BaseURLHost] == nil {
			byHost[s.BaseURLHost] = set{}
		}
		byHost[s.BaseURLHost][s.SecretFingerprint] = struct{}{}
	}
	out := map[string]bool{}
	for host, fps := range byHost {
		if len(fps) > 1 {
			out[host] = true
		}
	}
	return out
}
