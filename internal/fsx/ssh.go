package fsx

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/oldwinter/hctl/internal/exitcode"
)

// runTimeout bounds every SSH invocation so doctor/LookPath cannot hang.
var runTimeout = 10 * time.Second

// Runner runs a command. stdin may be nil.
type Runner func(stdin []byte, name string, args ...string) (stdout []byte, err error)

// SSH is a remote filesystem implemented with the OpenSSH client.
type SSH struct {
	Target   string
	Identity string
	Run      Runner
}

func (s SSH) args(remoteCmd string) []string {
	out := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=8",
		"-o", "StrictHostKeyChecking=accept-new",
	}
	if s.Identity != "" {
		out = append(out, "-i", s.Identity)
	}
	out = append(out, s.Target, "--", "sh", "-c", remoteCmd)
	return out
}

func (s SSH) run(stdin []byte, remoteCmd string) ([]byte, error) {
	run := s.Run
	if run == nil {
		run = defaultRunner
	}
	args := s.args(remoteCmd)
	out, err := run(stdin, "ssh", args...)
	if err != nil {
		return out, exitcode.Wrap(exitcode.SSH, fmt.Errorf("ssh %s: %w", s.Target, err))
	}
	return out, nil
}

func defaultRunner(stdin []byte, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return stdout.Bytes(), fmt.Errorf("ssh timed out after %s: %w", runTimeout, ctx.Err())
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return stdout.Bytes(), fmt.Errorf("%v: %s", err, msg)
		}
		return stdout.Bytes(), err
	}
	return stdout.Bytes(), nil
}

// LookPath runs `command -v` on the remote host. It does not probe --version.
func (s SSH) LookPath(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, " \t\n;$`|&<>(){}") {
		return "", fmt.Errorf("invalid binary name %q", name)
	}
	out, err := s.run(nil, fmt.Sprintf(`command -v %s`, shq(name)))
	if err != nil {
		return "", err
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return "", os.ErrNotExist
	}
	return p, nil
}

func (s SSH) Join(elem ...string) string {
	return path.Join(elem...)
}

func (s SSH) ReadFile(name string) ([]byte, error) {
	out, err := s.run(nil, fmt.Sprintf(`if [ -f %s ]; then cat %s; else exit 3; fi`, shq(name), shq(name)))
	if err != nil && isExit(err, 3) {
		return nil, os.ErrNotExist
	}
	return out, err
}

func (s SSH) WriteFile(name string, data []byte, perm os.FileMode) error {
	mode := fmt.Sprintf("%04o", perm&0o777)
	_, err := s.run(data, fmt.Sprintf(`umask 077; mkdir -p %s; cat > %s; chmod %s %s`, shq(path.Dir(name)), shq(name), mode, shq(name)))
	return err
}

// WriteNewFile fails when name exists: the -e/-L check reports it as exit 17
// (EEXIST) and set -C noclobber backstops any symlink planted after the check.
func (s SSH) WriteNewFile(name string, data []byte, perm os.FileMode) error {
	mode := fmt.Sprintf("%04o", perm&0o777)
	_, err := s.run(data, fmt.Sprintf(`umask 077; if [ -e %s ] || [ -L %s ]; then exit 17; fi; set -C; cat > %s && chmod %s %s`,
		shq(name), shq(name), shq(name), mode, shq(name)))
	if err != nil && isExit(err, 17) {
		return fmt.Errorf("%s: %w", name, fs.ErrExist)
	}
	return err
}

func (s SSH) Stat(name string) (os.FileInfo, error) {
	out, err := s.run(nil, fmt.Sprintf(`if [ -e %s ]; then if [ -d %s ]; then echo dir; else echo file; fi; stat -c %%s %s 2>/dev/null || stat -f %%z %s; else exit 3; fi`, shq(name), shq(name), shq(name), shq(name)))
	if err != nil && isExit(err, 3) {
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	dir := len(lines) > 0 && lines[0] == "dir"
	var size int64
	if len(lines) > 1 {
		size, _ = strconv.ParseInt(strings.TrimSpace(lines[1]), 10, 64)
	}
	return sshInfo{name: path.Base(name), dir: dir, size: size}, nil
}

func (s SSH) MkdirAll(name string, perm os.FileMode) error {
	_, err := s.run(nil, fmt.Sprintf(`mkdir -p %s`, shq(name)))
	return err
}

func (s SSH) Rename(oldpath, newpath string) error {
	_, err := s.run(nil, fmt.Sprintf(`mv %s %s`, shq(oldpath), shq(newpath)))
	return err
}

func (s SSH) Remove(name string) error {
	_, err := s.run(nil, fmt.Sprintf(`rm -f %s`, shq(name)))
	return err
}

// RemoteHome asks the remote shell for $HOME.
func (s SSH) RemoteHome() (string, error) {
	out, err := s.run(nil, `printf %s "$HOME"`)
	if err != nil {
		return "", err
	}
	home := strings.TrimSpace(string(out))
	if home == "" {
		return "", exitcode.Errorf(exitcode.SSH, "remote HOME is empty")
	}
	return home, nil
}

func shq(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func isExit(err error, code int) bool {
	var ee *exec.ExitError
	if !asExit(err, &ee) {
		// exitcode.Error unwraps
		var unwrap interface{ Unwrap() error }
		if u, ok := err.(interface{ Unwrap() error }); ok {
			return isExit(u.Unwrap(), code)
		}
		_ = unwrap
		return strings.Contains(err.Error(), "exit status "+strconv.Itoa(code))
	}
	return ee.ExitCode() == code
}

func asExit(err error, target **exec.ExitError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*exec.ExitError); ok {
		*target = e
		return true
	}
	if u, ok := err.(interface{ Unwrap() error }); ok {
		return asExit(u.Unwrap(), target)
	}
	return false
}

type sshInfo struct {
	name string
	dir  bool
	size int64
}

func (s sshInfo) Name() string { return s.name }
func (s sshInfo) Size() int64  { return s.size }
func (s sshInfo) Mode() os.FileMode {
	if s.dir {
		return os.ModeDir | 0o755
	}
	return 0o600
}
func (s sshInfo) ModTime() time.Time { return time.Time{} }
func (s sshInfo) IsDir() bool        { return s.dir }
func (s sshInfo) Sys() any           { return nil }
