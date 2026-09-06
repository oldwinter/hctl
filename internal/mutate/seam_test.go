package mutate

import (
	"errors"
	"testing"
)

func TestDefaultBackupDirNoHome(t *testing.T) {
	old := userHomeDir
	defer func() { userHomeDir = old }()
	t.Setenv("HARNESSCTL_BACKUP_DIR", "")
	userHomeDir = func() (string, error) { return "", errors.New("no") }
	if DefaultBackupDir("") == "" {
		t.Fatal("fallback")
	}
}
