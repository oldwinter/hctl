package adapters

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/oldwinter/hctl/internal/fsx"
)

// pathLooker resolves a binary name on a filesystem (local PATH or remote command -v).
type pathLooker interface {
	LookPath(name string) (string, error)
}

type commandProbePolicy interface {
	CommandProbesAllowed() bool
}

// DetectBinaryFS resolves binaries on fsys.
// SSH contexts use remote `command -v` only — no remote --version (hang risk).
func DetectBinaryFS(fsys fsx.FS, names []string) (path string, version string, ok bool) {
	lp, hasLP := fsys.(pathLooker)
	_, remote := fsys.(fsx.SSH)
	allowCommandProbes := !remote
	if policy, ok := fsys.(commandProbePolicy); ok {
		allowCommandProbes = policy.CommandProbesAllowed()
	}
	if !hasLP {
		return "", "", false
	}
	for _, name := range names {
		p, err := lp.LookPath(name)
		if err != nil || strings.TrimSpace(p) == "" {
			continue
		}
		ver := ""
		if allowCommandProbes {
			ver = ProbeVersion(p)
		}
		return p, ver, true
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
