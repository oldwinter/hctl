# Changelog

All notable changes to this project are documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning: [SemVer](https://semver.org/).

## [Unreleased]

### Added
- `hctl list` is an alias of `hctl get`.
- Root help names the first commands. `get harnesses`, `doctor`, and `describe harness NAME`.
- Config identity provenance: `--config` / env / existing `~/.hctl/config.yaml` / existing `~/.harnessctl/config.yaml` / else create `~/.hctl/config.yaml`. Writes stay in place; legacy files are never copied or deleted.
- Preferred env names `HCTL_HOME`, `HCTL_CONFIG`, `HCTL_BACKUP_DIR`, `HCTL_SSH` (legacy `HARNESSCTL_*` still works).

### Changed
- `remote.Dial` is the only home resolver. Dead `ResolveHome` / local-only adapter `Read(home)` / `FSReader` fallbacks are gone.
- Adapter `Read` always takes `(fs, home)`. Inventory and writes share one desired-field table (`model.ChangesFromDesired`).
- Already-converged `apply` / `set` skip backup and write. `config.Save` is backup → temp → rename → re-read.
- `diff -f` rejects unknown harness names the same way `apply` does.
- Claude and Grok refuse `set provider` in `ValidateDesired`, not in a mutate special case.

### Fixed
- `hctl sync` without `--from`/`--to` exits 2 and prints a `--dry-run` example.
- `hctl config use-context` without NAME exits 2 and names `get-contexts`.
- Default `sync --fields model,provider` skips provider on claude/grok instead of failing the batch.
- `hctl config set-context` without NAME exits 2 and prints an ssh example.
- `hctl diff` without a resource prints a `--home-a` / `--home-b` example.
- Empty `hctl config get-contexts` names `set-context` instead of a header-only table.
- Help text leads with `hctl` (legacy `HARNESSCTL_*` / `~/.harnessctl` names mentioned once for compatibility).
- `hctl get` without a resource names valid resources and exits with usage code 2.
- `hctl version` always prints version, commit, and build date (`just build` / `just release` inject via ldflags).
- `set --dry-run` PATH column shows the config path that would be written.
- README / CONTRIBUTING install docs match the populated GitHub repo.
- `apply`, `diff`, `sync`, and `completion` help examples use `hctl`, not `harnessctl`.
- `describe`, `set`, `completion`, and `apply` missing args exit 2 and print an example.
- `hctl version` prints a `just build` hint when commit or date is `unknown`.
- `hctl doctor` names `hctl describe harness NAME` after a parse error.
- `hctl config current-context` prints the resolved config path. `--json` adds `config`.

## [1.0.1] - 2026-09-06

Hardening after the 1.0.0 report. Module path no longer collides with the Rust `harnessctl`.

### Changed
- Module is `github.com/oldwinter/hctl`. Primary binary is `hctl`; `harnessctl` remains a secondary entrypoint.
- SSH `doctor` install checks use remote `command -v` (10s timeout), not the local PATH. Remote `--version` is not probed.
- Claude onboarding reads `~/.claude.json` (`theme` + `hasCompletedOnboarding`). Missing or incomplete first-run is `onboarding=needed`.
- Key-drift messages name the grouped harnesses: `key drift on host HOST (claude,codex)`.
- `set provider` for Claude and Grok returns a usage error (including `--dry-run`) instead of a silent no-op.
- `just release` writes `dist/hctl_linux_amd64`, `dist/harnessctl_linux_amd64`, and `dist/SHA256SUMS`.

### Security
- Same secret redaction contract as 1.0.0.

### Known limitations
- JSONC comments and trailing commas are dropped on write (emitted as JSON).
- JSON key order may change after `set`/`apply`.
- TOML/YAML comments on **unrelated** keys are kept; new keys are appended.
- cursor-agent login probe still uses local `cursor-agent status` (800ms) and is skipped on SSH filesystems.
- 1.0 no brew tap / GUI.

## [1.0.0] - 2026-09-06

First production-ready release.

### Added
- Write path: `set model|provider` with `--dry-run`, atomic write, backup, post-write verify.
- Desired state: `apply -f` / `diff -f` for TOML or YAML.
- SSH contexts via OpenSSH (`BatchMode`, `ConnectTimeout`, optional `IdentityFile`).
- `sync --from --to --harness --fields` (model, provider, secret-ref, optional bearer copy).
- Adapters for `pi`, `droid`/`factory`, `cursor`/`cursor-agent`.
- Doctor key-drift (same base URL host, different fingerprints).
- `config set-context`, `--output wide`, shell completions.
- GitHub Actions CI (`test`, `vet`, `build`).
- Stable exit codes: 2 usage, 3 verify, 4 ssh, 5 parse.

### Security
- Snapshots never store or print plaintext keys.
- Bearer sync copies bytes without logging; only fingerprints are shown.

## [0.3.0] - 2026-09-06

Desired-state apply/diff.

## [0.2.0] - 2026-09-06

Write path foundation.

## [0.1.0] - 2026-09-06

Read-only inventory, diff, doctor, local contexts.
