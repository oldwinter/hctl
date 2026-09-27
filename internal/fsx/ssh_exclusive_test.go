package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/oldwinter/hctl/internal/testutil"
)

func TestSSHWriteNewFileRejectsDirectoryEntries(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(fmt.Sprintf("symlink=%v", symlink), func(t *testing.T) {
			root := t.TempDir()
			foreign := filepath.Join(root, "foreign")
			if err := os.Mkdir(foreign, 0o700); err != nil {
				t.Fatal(err)
			}
			entry := foreign
			if symlink {
				entry = filepath.Join(root, "directory-link")
				if err := os.Symlink(foreign, entry); err != nil {
					t.Fatal(err)
				}
			}
			s := SSH{Target: "user@box", Run: testutil.SSHLocalRunner(t)}
			if err := s.WriteNewFile(entry, []byte("new-config"), 0o600); !errors.Is(err, fs.ErrExist) {
				t.Errorf("directory collision = %v, want ErrExist", err)
			}
			if entries, err := os.ReadDir(foreign); err != nil || len(entries) != 0 {
				t.Errorf("foreign directory modified: %v (%v)", entries, err)
			}
			if _, err := os.Lstat(entry); err != nil {
				t.Errorf("competing entry removed: %v", err)
			}
			if _, err := os.Stat(entry + ".d"); !os.IsNotExist(err) {
				t.Errorf("reservation directory left behind: %v", err)
			}
		})
	}
}

func TestSSHAtomicWriteRejectsRacingDirectoryEntries(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		for _, existing := range []bool{false, true} {
			t.Run(fmt.Sprintf("symlink=%v/existing=%v", symlink, existing), func(t *testing.T) {
				root := t.TempDir()
				foreign := filepath.Join(root, "foreign")
				if err := os.Mkdir(foreign, 0o700); err != nil {
					t.Fatal(err)
				}
				dest := filepath.Join(root, "config.json")
				if existing {
					if err := os.WriteFile(dest, []byte("old-config"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				var planted string
				s := SSH{Target: "user@box", Run: plantingRunner(t, root, func(tmp string) string {
					planted = tmp
					if symlink {
						return fmt.Sprintf("ln -s %s %s", shq(foreign), shq(tmp))
					}
					foreign = tmp
					return "mkdir " + shq(tmp)
				})}
				if err := AtomicWrite(s, dest, []byte("new-config"), 0o600); err != nil {
					t.Errorf("atomic write must retry collision: %v", err)
				}
				if planted == "" {
					t.Fatal("collision was not injected")
				}
				if st, err := os.Lstat(planted); err != nil || (st.Mode()&os.ModeSymlink != 0) != symlink {
					t.Errorf("competing entry changed: %v (%v)", st, err)
				}
				if entries, err := os.ReadDir(foreign); err != nil || len(entries) != 0 {
					t.Errorf("foreign directory modified: %v (%v)", entries, err)
				}
				if st, err := os.Lstat(dest); err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0o600 {
					t.Errorf("destination must be a private regular file: %v (%v)", st, err)
				}
				if data, err := os.ReadFile(dest); err != nil || string(data) != "new-config" {
					t.Errorf("destination payload mismatch: %v", err)
				}
			})
		}
	}
}
