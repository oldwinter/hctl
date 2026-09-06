# Changelog

All notable changes to this project are documented in this file.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning: [SemVer](https://semver.org/).

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

### Known limitations
- JSONC comments and trailing commas are dropped on write (emitted as JSON).
- JSON key order may change after `set`/`apply`.
- TOML/YAML comments on **unrelated** keys are kept; new keys are appended.
- Remote install detection uses the local PATH for binaries; SSH `doctor` reports remote config, not remote `LookPath` (unless you run the binary on that host).
- Grok/Claude `set provider` is inferred or implicit; verify skips those provider fields.

## [0.3.0] - 2026-09-06

Desired-state apply/diff.

## [0.2.0] - 2026-09-06

Write path foundation.

## [0.1.0] - 2026-09-06

Read-only inventory, diff, doctor, local contexts.
