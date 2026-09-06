package remote

import (
	"testing"
	"time"

	"github.com/oldwinter/hctl/internal/config"
)

func TestDefaultDialSSHRemoteHomeFailsFast(t *testing.T) {
	t.Setenv("HARNESSCTL_SSH", "")
	done := make(chan error, 1)
	go func() {
		_, _, err := DefaultDial(config.NamedContext{
			Name:    "ssh",
			Context: config.Context{Kind: config.KindSSH, SSH: "nobody@127.0.0.1:1"},
		}, "")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected remote home/ssh error")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timeout")
	}
}
