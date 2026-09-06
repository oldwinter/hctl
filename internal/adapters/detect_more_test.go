package adapters

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestProbeVersionEmptyAndTruncate(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "nobin")
	if runtime.GOOS != "windows" {
		os.WriteFile(bin, []byte("#!/bin/sh\nexit 1\n"), 0o755)
		_ = ProbeVersion(bin)
		os.WriteFile(bin, []byte("#!/bin/sh\necho\n"), 0o755)
		_ = ProbeVersion(bin)
		long := filepath.Join(dir, "long")
		os.WriteFile(long, []byte("#!/bin/sh\npython3 -c 'print(\"x\"*200)' 2>/dev/null || printf '%0200d' 0\n"), 0o755)
		_ = ProbeVersion(long)
	}
	if truncate("abc", 2) != "ab" {
		t.Fatal(truncate("abc", 2))
	}
	_, _, ok := DetectBinary([]string{"this-really-does-not-exist-zz"})
	if ok {
		t.Fatal("should miss")
	}
}
