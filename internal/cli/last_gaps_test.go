package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oldwinter/hctl/internal/config"
	"github.com/oldwinter/hctl/internal/fsx"
	"github.com/oldwinter/hctl/internal/remote"
	"github.com/oldwinter/hctl/internal/testutil"
)

type boomReadFS struct{ fsx.Local }

func (boomReadFS) ReadFile(string) ([]byte, error) { return nil, errors.New("read boom") }

func TestDiffSideBReadFail(t *testing.T) {
	dir := t.TempDir()
	homeA := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	homeB := testutil.CopyTree(t, testutil.Testdata(t, "home-b"))
	cfgPath := filepath.Join(dir, "c.yaml")
	f := config.Default()
	f.Contexts = []config.NamedContext{
		{Name: "a", Context: config.Context{Kind: config.KindLocal, Home: homeA}},
		{Name: "b", Context: config.Context{Kind: config.KindLocal, Home: homeB}},
	}
	f.CurrentContext = "a"
	config.Save(cfgPath, f)
	old := remote.Dial
	t.Cleanup(func() { remote.Dial = old })
	remote.Dial = func(nc config.NamedContext, homeFlag string) (fsx.FS, string, error) {
		h := nc.Context.Home
		if homeFlag != "" {
			h = homeFlag
		}
		if nc.Name == "b" {
			return boomReadFS{}, h, nil
		}
		return fsx.Local{}, h, nil
	}
	if _, err := run(t, "--config", cfgPath, "diff", "--contexts", "a,b", "harness", "codex"); err == nil {
		t.Fatal("expected side b fail")
	}
}

func TestSyncApplyProviderRejectAndSecretErrAndRef(t *testing.T) {
	dir := t.TempDir()
	homeA := testutil.CopyTree(t, testutil.Testdata(t, "home-a"))
	homeB := testutil.CopyTree(t, testutil.Testdata(t, "home-b"))
	cfgPath := filepath.Join(dir, "c.yaml")
	f := config.Default()
	f.Contexts = []config.NamedContext{
		{Name: "a", Context: config.Context{Kind: config.KindLocal, Home: homeA}},
		{Name: "b", Context: config.Context{Kind: config.KindLocal, Home: homeB}},
	}
	f.CurrentContext = "a"
	config.Save(cfgPath, f)

	// Apply error: sync provider onto claude (unsupported)
	if _, err := run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "claude", "--fields", "provider"); err == nil {
		// only errors if src has Provider non-empty; force via set on a copy
		t.Log("claude provider sync:", err)
	}
	// ensure claude has provider in snapshot — write settings
	os.MkdirAll(filepath.Join(homeA, ".claude"), 0o755)
	os.WriteFile(filepath.Join(homeA, ".claude", "settings.json"), []byte(`{"model":"m","provider":"anthropic"}`), 0o600)
	if _, err := run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "claude", "--fields", "provider"); err == nil {
		t.Fatal("expected apply reject")
	}

	// CopySecret error via AtomicWrite fail
	prev := fsx.AtomicWriteHook
	t.Cleanup(func() { fsx.AtomicWriteHook = prev })
	fsx.AtomicWriteHook = func(fsys fsx.FS, path string, data []byte, mode os.FileMode) error {
		return errors.New("aw")
	}
	if _, err := run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "secret"); err == nil {
		t.Fatal("expected copy secret fail")
	}
	fsx.AtomicWriteHook = prev

	// Ref print path: successful secret copy with non-empty Ref
	os.WriteFile(filepath.Join(homeA, ".codex", "config.toml"), []byte("model = \"gpt-5.2-codex\"\nmodel_provider = \"custom\"\n\n[model_providers.custom]\nname = \"Custom Gateway\"\nbase_url = \"https://api.openai.com/v1\"\nexperimental_bearer_token = \"sk-test-aaa\"\nenv_key = \"OPENAI_API_KEY\"\nwire_api = \"responses\"\n"), 0o600)
	out, err := run(t, "--config", cfgPath, "sync", "--from", "a", "--to", "b", "--harness", "codex", "--fields", "secret")
	if err != nil {
		t.Fatal(err, out)
	}
	if !strings.Contains(out, "ref=") {
		t.Fatalf("expected ref in output: %q", out)
	}
}
