# Changelog

All notable changes to this project are documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning: [SemVer](https://semver.org/).

## [Unreleased]

### Fixed
- Help text leads with `hctl` (legacy `HARNESSCTL_*` / `~/.harnessctl` names mentioned once for compatibility).
- `hctl get` without a resource names valid resources and exits with usage code 2.
- `hctl version` always prints version, commit, and build date (`just build` / `just release` inject via ldflags).
- `set --dry-run` PATH column shows the config path that would be written.
- README / CONTRIBUTING install docs match the populated GitHub repo.

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
