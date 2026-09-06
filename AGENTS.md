# AGENTS.md

`harnessctl` is a **read-only (v0.1)** kubectl-style CLI for coding-agent harness **configuration**.

## Boundaries

- **This repo** inventories and diffs harness configs, default models, and providers across environments (`mba`, `box`).
- **Does not dispatch agents.** That is `herdr-orchestrator`. Do not add a runner, session broker, or task queue here.
- **Does not replace `all-cli`.** Do not absorb shell aliases, prompt kits, or unrelated developer utilities.
- **Does not write harness configs in v0.1.** No `set` / `apply` / `sync`. Parsers are read-only.
- **Never print plaintext API keys or tokens.** Snapshots store a SHA-256 fingerprint (first 8 hex chars) or an env-var reference. Table / `String()` / JSON go through a redact guard.

## Layout

- `cmd/harnessctl`, `cmd/hctl` — binaries (same CLI)
- `internal/cli` — Cobra commands
- `internal/config` — `~/.harnessctl/config.yaml` (contexts)
- `internal/model` — unified snapshot
- `internal/adapters/<harness>` — on-disk readers
- `internal/render` — table / JSON
- `testdata/` — fake keys only (`sk-test-aaa`, `sk-test-bbb`)

## v0.2 (out of scope here)

SSH context execution, and write paths (`set` / `apply` / `sync`).
