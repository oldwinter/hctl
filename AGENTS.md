# AGENTS.md

`harnessctl` is the kubectl-style **config** control plane for coding-agent harnesses.

## Boundaries

- Manages harness **config** (get/describe/diff/set/apply/sync/doctor) across `mba` / `box`.
- **Does not dispatch agents.** That is herdr-orchestrator.
- **Does not replace all-cli.**
- **Never print plaintext API keys.** Fingerprint or env-ref only. Bearer sync copies bytes without logging.
- Fixtures use `sk-test-aaa` / `sk-test-bbb` only.

## Write path

Backup → temp file → rename → re-read verify. JSONC comments are dropped on write (documented).

## Layout

`cmd/`, `internal/cli`, `internal/config`, `internal/model`, `internal/adapters/<harness>`, `internal/edit`, `internal/fsx`, `internal/mutate`, `internal/remote`, `internal/render`, `testdata/`.
