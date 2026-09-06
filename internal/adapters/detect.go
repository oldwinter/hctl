package adapters

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// DetectBinary looks up the first name on PATH and optionally probes a version.
func DetectBinary(names []string) (path string, version string, ok bool) {
	for _, name := range names {
		p, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		return p, ProbeVersion(p), true
	}
	return "", "", false
}

// ProbeVersion tries a few common version flags with a short timeout.
func ProbeVersion(bin string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	for _, args := range [][]string{{"version"}, {"--version"}, {"-v"}} {
		cmd := exec.CommandContext(ctx, bin, args...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		if err := cmd.Run(); err != nil {
			if ctx.Err() != nil {
				return ""
			}
			continue
		}
		line := firstLine(buf.String())
		if line != "" {
			return truncate(line, 80)
		}
	}
	return ""
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
