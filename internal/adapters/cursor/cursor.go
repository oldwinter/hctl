package cursor

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/oldwinter/hctl/internal/edit"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/model"
)

// Adapter reads ~/.cursor/cli-config.json and optionally probes cursor-agent status.
type Adapter struct{}

func (Adapter) Name() string          { return "cursor-agent" }
func (Adapter) Aliases() []string     { return []string{"cursor"} }
func (Adapter) BinaryNames() []string { return []string{"cursor-agent", "cursor"} }
func (Adapter) ConfigRelPaths() []string {
	return []string{".cursor/cli-config.json"}
}

type cliConfig struct {
	Model    string `json:"model"`
	Provider string `json:"provider"`
}

type commandProbePolicy interface {
	CommandProbesAllowed() bool
}

func (a Adapter) Read(home string) (model.Snapshot, error) {
	return a.ReadFS(fsx.Local{}, home)
}

func (a Adapter) ReadFS(fsys fsx.FS, home string) (model.Snapshot, error) {
	path := fsys.Join(home, ".cursor", "cli-config.json")
	snap := model.Snapshot{Name: a.Name(), ConfigPaths: []string{path}}
	data, err := fsys.ReadFile(path)
	if err != nil {
		if fsx.IsNotExist(err) {
			return snap, nil
		}
		return snap, err
	}
	snap.ConfigFound = true
	var cfg cliConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		snap.ParseError = err.Error()
		return snap, nil
	}
	snap.DefaultModel = cfg.Model
	snap.Provider = cfg.Provider
	allowProbe := true
	if policy, ok := fsys.(commandProbePolicy); ok {
		allowProbe = policy.CommandProbesAllowed()
	}
	if _, local := fsys.(fsx.Local); local && allowProbe {
		if logged, note := probeLogin(); note != "" {
			snap.Notes = append(snap.Notes, note)
			if !logged {
				snap.Notes = append(snap.Notes, "cursor-agent login not detected")
			}
		}
	}
	return snap, nil
}

func probeLogin() (loggedIn bool, note string) {
	bin, err := exec.LookPath("cursor-agent")
	if err != nil {
		return false, ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "status")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return false, "cursor-agent status timed out"
	}
	if err != nil {
		return false, ""
	}
	s := strings.ToLower(string(out))
	if strings.Contains(s, "logged") || strings.Contains(s, "authenticated") {
		return true, "cursor-agent status: logged in"
	}
	if strings.Contains(s, "login") || strings.Contains(s, "unauth") {
		return false, "cursor-agent status: not logged in"
	}
	return false, ""
}

func (a Adapter) WriteFields(fsys fsx.FS, home string, d model.Desired) ([]string, error) {
	path := fsys.Join(home, ".cursor", "cli-config.json")
	data, err := fsx.ReadMaybe(fsys, path)
	if err != nil {
		return nil, err
	}
	if d.Model != "" {
		data, err = edit.SetJSON(data, []string{"model"}, d.Model)
		if err != nil {
			return nil, err
		}
	}
	if d.Provider != "" {
		data, err = edit.SetJSON(data, []string{"provider"}, d.Provider)
		if err != nil {
			return nil, err
		}
	}
	if err := fsx.AtomicWrite(fsys, path, data, 0o600); err != nil {
		return nil, err
	}
	return []string{path}, nil
}
