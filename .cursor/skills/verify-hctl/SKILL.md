---
name: verify-hctl
description: Drive the hctl CLI (kubectl-style harness config control plane; also built as harnessctl) against isolated testdata homes. Use when proving get/describe/doctor/set/apply/diff/sync/config behavior, exit codes, writes, or secret redaction — not unit tests.
---

# Verify hctl

`hctl` is a short-lived CLI. There is no server. Launch means build one isolated binary and a disposable home tree; each drive is its own process. The next agent reading this cold must never point the binary at the developer's real `$HOME`, `~/.hctl`, or `testdata/` originals.

Existing in-repo checks (`just smoke`, `scripts/verify-first-run.sh`, `go test ./internal/cli`) are complementary. This skill drives the **real binary** the way a user does.

## Launch

The helper builds into a per-run workspace and copies fixtures. Do not use `just smoke` as launch: it reads `testdata/home-a` in place.

```bash
.cursor/skills/verify-hctl/scripts/verify-hctl launch
```

Optional: `.cursor/skills/verify-hctl/scripts/verify-hctl launch --run-id <id>` when two agents share a machine.

Ready when stdout contains all of:

- `hctl version 1.0.1` (version string comes from `internal/cli/root.go`)
- `commit: <short SHA>` matching `git rev-parse --short HEAD` (not `unknown`)
- `READY: workspace /tmp/hctl-verify-<id>`
- `RUN_ID=<id>`

The workspace is `/tmp/hctl-verify-<id>/` and holds:

| Path | Role |
| --- | --- |
| `bin/hctl`, `bin/harnessctl` | binaries for this run only |
| `config.yaml` | verification kubeconfig: `mba`→home-a, `box`→home-b, `theme`→home-theme, `drift`→home-drift, `empty`→home-empty; all `kind: local` |
| `homes/home-*` | copies of `testdata/home-*` |
| `desired.toml` | copy of `testdata/desired.toml` |
| `backups/` | `$HCTL_BACKUP_DIR` for this run |
| `state.env` | sourced by later helper commands |

If `go build` fails, stop and fix the tree. Do not write around a broken build.

Teardown of a failed launch: `.cursor/skills/verify-hctl/scripts/verify-hctl cleanup <id>` if `READY` never printed but `/tmp/hctl-verify-<id>` exists, or `rm -rf /tmp/hctl-verify-<id>` only when that path matches the prefix.

Two runs can sit side by side: different `--run-id`s, different workspaces, different backup dirs. Never export a shared `HCTL_HOME` / `HCTL_CONFIG` / `HCTL_BACKUP_DIR` into the agent shell.

## Doctor

Read-only. Run this first whenever anything looks off:

```bash
.cursor/skills/verify-hctl/scripts/verify-hctl doctor
```

Pass the `RUN_ID` if more than one workspace exists. Worth driving only when every line is `ok` and the last line is `doctor: ready RUN_ID=...`.

Doctor checks, in order:

1. This run's `bin/hctl` and `bin/harnessctl` exist and are executable.
2. `hctl version` prints `hctl version 1.0.1` and the launch commit.
3. Workspace path is `/tmp/hctl-verify-*` (not the repo, not `$HOME`).
4. `homes/home-a` is a copy, not `testdata/home-a`.
5. `mba` Codex fixture is present (`homes/home-a/.codex/config.toml`).
6. `HCTL_SSH=0` inventory against this run's `--config` lists `codex` and `claude` and does not contain `sk-test-aaa` / `sk-test-bbb`.

Refuse to drive an instance that failed doctor. Do not fall back to the developer's `~/.codex` to "just see if it works".

## Drive

Drive only through the helper. It pins `--config` to the workspace file, sets `HCTL_SSH=0` and `HCTL_BACKUP_DIR`, and defaults to `--no-probe`.

```bash
.cursor/skills/verify-hctl/scripts/verify-hctl drive -- get harnesses
.cursor/skills/verify-hctl/scripts/verify-hctl drive --context box -- get harnesses
.cursor/skills/verify-hctl/scripts/verify-hctl drive --out inventory/get-harnesses.txt -- get harnesses
```

`drive` inserts `--config` and `--no-probe` before the args after `--`. Do **not** add `--home`: that flag overrides every context to one directory and collapses `mba`/`box` for `sync` / `diff --contexts`.

| User action | Helper command |
| --- | --- |
| List harnesses (default `mba` / home-a) | `drive -- get harnesses` |
| Same as `get` | `drive -- list harnesses` |
| One harness | `drive -- get harness codex` |
| Models table | `drive -- get models` |
| JSON models | `drive -- -o json get models` |
| Wide inventory (config paths) | `drive -- -o wide get harnesses` |
| Describe | `drive -- describe harness codex` |
| Product doctor on mba | `drive -- doctor` |
| Product doctor on theme leftover | `drive --context theme -- doctor` |
| Product doctor on key-drift | `drive --context drift -- doctor` |
| Empty home hint | `drive --context empty -- get harnesses` |
| Contexts | `drive -- config get-contexts` |
| Current context | `drive -- config current-context` |
| Switch context | `drive -- config use-context box` |
| Add a context | `drive -- config set-context lab --kind local --home /tmp/unused-hctl-lab` |
| Diff two fixture homes | `drive -- diff harness codex --contexts mba,box` |
| Desired vs current | `drive -- diff -f "$HCTL_VERIFY_DESIRED"` (path from `status`) |
| Dry-run set | `drive -- set model codex o4-mini --dry-run` |
| Write set | `drive -- set model codex o4-mini` |
| Unsupported provider | `drive -- set provider claude custom --dry-run` (usage error) |
| Dry-run apply | `drive -- apply -f <desired> --dry-run` |
| Apply | `drive -- apply -f <desired>` |
| Dry-run sync | `drive -- sync --from mba --to box --harness codex --fields model --dry-run` |
| Sync | `drive -- sync --from mba --to box --harness codex --fields model` |
| Completions | `drive -- completion bash` |
| Secondary binary | `"$HCTL_VERIFY_ALIAS" version` after `status` (same flags as `hctl`) |

Need env paths (`HCTL_VERIFY_DESIRED`, homes, backup dir):

```bash
.cursor/skills/verify-hctl/scripts/verify-hctl status
```

`--probe` opts into version / `cursor-agent status` subprocesses. Leave it off unless the feature under test is install detection.

Stable handles: command names, resource names (`harnesses`, `models`, `harness`), harness IDs (`codex`, `claude`, `grok`, `hermes`, `opencode`, `pi`, `droid`, `cursor-agent`), context names (`mba`, `box`, `theme`, `drift`, `empty`), table headers (`NAME`, `HARNESS`, `FIELD`, `CURRENT`), and the literals `dry-run: no files written`, `verified: ok`, `Next: hctl get harnesses -o wide`.

Home-a Codex starts at `model = "gpt-5.2-codex"` / provider `custom`. Home-b Codex starts at `model = "o4-mini"`. Claude on home-a is `claude-sonnet-4`. `testdata/desired.toml` wants Codex `o4-mini` and Claude `claude-opus-4`.

Exit codes: `0` ok, `1` generic, `2` usage, `3` write-verify failed, `4` SSH (should not appear; SSH is disabled), `5` parse. Missing `apply -f`, `describe` args, `sync --from/--to`, and unknown `-o yaml` are usage `2`.

Read the feature map before choosing a path. A proof that only hits `get harnesses` is incomplete when the map lists `get harness NAME`, `-o json`, and `describe`.

## Evidence

Durable root: `.cursor/skills/verify-hctl/artifacts/` (survives cleanup). Name files `<feature>/<step>.txt`.

Every proof must include:

1. **Action** — the exact `verify-hctl drive` line and the captured command/exit in the `--out` file.
2. **Visible result** — stdout/stderr (table, `verified: ok`, usage text).
3. **Side effect** — for writes: a second read (`describe` / `get models`) **and** the on-disk field (`grep '^model =' "$HCTL_VERIFY_HOME_A/.codex/config.toml"`). For dry-run: the same file is **unchanged** after the dry-run. For `set`/`apply` success: a new file under `$HCTL_VERIFY_BACKUP` whose name contains the harness and source basename.
4. **Redaction** — stdout/stderr/JSON must not contain `sk-test-aaa` or `sk-test-bbb`. SECRET column is `sha256:<8 hex>` or `env:<name>`. Hosts have no userinfo. The helper already fails the drive on those two fixture keys; still fail the proof if any `sk-` token appears.

Standards:

- Drive the real binary and user flags. Do not treat `go test` or `mutate.Apply(...)` as a user-path proof.
- Capture the command and the resulting state, not only the last screen.
- `--dry-run` is proven by an unchanged file (and no new backup), not by the word "dry-run" alone.
- Do not invent a live SSH `box`. The verification `box` context is a **local copy** of `testdata/home-b`. `HCTL_SSH=0` must stay set so a mistaken SSH context cannot leave the machine.
- Do not print or copy bearer bytes into artifacts. `--fields secret` is out of scope unless a feature file explicitly asks for fingerprint-only output.

## Cleanup

```bash
.cursor/skills/verify-hctl/scripts/verify-hctl cleanup
```

Removes only `/tmp/hctl-verify-<id>/` (the instance this run created). It does not kill processes by name, does not delete `.cursor/skills/verify-hctl/artifacts/`, and does not touch `testdata/` or the developer's `$HOME`.

After cleanup, confirm the `--out` files still exist under `artifacts/`. If they are gone, cleanup is wrong — stop and fix the skill before reporting a pass.

## Helpers

All helper invocations use the executable at `.cursor/skills/verify-hctl/scripts/verify-hctl`:

```bash
.cursor/skills/verify-hctl/scripts/verify-hctl launch
.cursor/skills/verify-hctl/scripts/verify-hctl doctor
.cursor/skills/verify-hctl/scripts/verify-hctl drive --out inventory/get-harnesses.txt -- get harnesses
.cursor/skills/verify-hctl/scripts/verify-hctl status
.cursor/skills/verify-hctl/scripts/verify-hctl cleanup
```

There is no other control script. Do not reverse-engineer flags from the script when this file already names them.
