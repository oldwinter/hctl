package adapters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oldwinter/hctl/internal/model"
)

// Doctor derives check rows from snapshots, including key-drift by shared host.
func Doctor(snaps []model.Snapshot) []model.DoctorCheck {
	driftNames := keyDriftNamesByHost(snaps)
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
			} else {
				inspect := fmt.Sprintf("Inspect: hctl describe harness %s", s.Name)
				if c.Message == "" {
					c.Message = inspect
				} else {
					c.Message = c.Message + "; " + inspect
				}
			}
		} else {
			c.Onboarding = "ok"
		}
		if names, ok := driftNames[s.BaseURLHost]; ok && s.BaseURLHost != "" {
			c.Drift = "key-drift"
			msg := fmt.Sprintf("key drift on host %s (%s)", s.BaseURLHost, strings.Join(names, ","))
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

func keyDriftNamesByHost(snaps []model.Snapshot) map[string][]string {
	type set map[string]struct{}
	fps := map[string]set{}
	names := map[string][]string{}
	seen := map[string]map[string]bool{}
	for _, s := range snaps {
		if s.BaseURLHost == "" || s.SecretFingerprint == "" {
			continue
		}
		if fps[s.BaseURLHost] == nil {
			fps[s.BaseURLHost] = set{}
			seen[s.BaseURLHost] = map[string]bool{}
		}
		fps[s.BaseURLHost][s.SecretFingerprint] = struct{}{}
		if !seen[s.BaseURLHost][s.Name] {
			names[s.BaseURLHost] = append(names[s.BaseURLHost], s.Name)
			seen[s.BaseURLHost][s.Name] = true
		}
	}
	out := map[string][]string{}
	for host, fpset := range fps {
		if len(fpset) > 1 {
			n := append([]string(nil), names[host]...)
			sort.Strings(n)
			out[host] = n
		}
	}
	return out
}
