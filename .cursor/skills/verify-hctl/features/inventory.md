# Inventory

Inventory lets a user list harness configs and default models, inspect one harness, and see the next command when a home has no configs.

## Sub-features

- `inventory-list` lists every known harness from the current context home.
- `inventory-list-alias` accepts `list` as the same command as `get`.
- `inventory-one` shows a single harness row and does not list the others.
- `inventory-models` lists default models for each harness.
- `inventory-json` emits JSON objects (array when listing, object when named).
- `inventory-wide` adds a CONFIG column with fixture paths under the isolated home.
- `inventory-describe` prints a kubectl-like block for one harness.
- `inventory-empty` on a home with no configs prints `Next: hctl get harnesses -o wide`.

## How to get to it (user POV)

- Run `hctl get harnesses` or `hctl list harnesses`.
- Run `hctl get harnesses -o json` or `hctl get harnesses -o wide`.
- Run `hctl get harness codex` (NAME works for `models` too).
- Run `hctl get models` or `hctl get models -o json`.
- Run `hctl describe harness codex`.
- Run the same commands against a home that has no harness files (`empty` context).

## Driving it with verify-hctl

Preconditions:

- `verify-hctl doctor` reports ready.
- Current context is `mba` (home-a copy). Codex model is `gpt-5.2-codex`. Claude model is `claude-sonnet-4`.
- The `empty` context home has no `.codex` / `.claude` files.

- **List harnesses.** Run `.cursor/skills/verify-hctl/scripts/verify-hctl drive --out inventory/get-harnesses.txt -- get harnesses`. Exit `0`. Stdout starts with `NAME` and includes rows `codex`, `claude`, `grok`, `hermes`, `opencode`, `pi`, `droid`, `cursor-agent`. Codex MODEL is `gpt-5.2-codex`. SECRET is `sha256:` plus 8 hex, not a raw key.
- **List alias.** Run `verify-hctl drive --out inventory/list-harnesses.txt -- list harnesses`. Exit `0`. The NAME set matches `get harnesses`.
- **One harness.** Run `verify-hctl drive --out inventory/get-harness-codex.txt -- get harness codex`. Exit `0`. Stdout contains `codex` and `gpt-5.2-codex` and does not contain a `claude` row.
- **Models table.** Run `verify-hctl drive --out inventory/get-models.txt -- get models`. Exit `0`. Header is `HARNESS PROVIDER MODEL …`. Codex MODEL is `gpt-5.2-codex`. Claude MODEL is `claude-sonnet-4`.
- **JSON models.** Run `verify-hctl drive --out inventory/get-models.json.txt -- -o json get models`. Exit `0`. Stdout is a JSON array containing `"harness": "codex"` and `"model": "gpt-5.2-codex"`.
- **Wide paths.** Run `verify-hctl drive --out inventory/get-harnesses-wide.txt -- -o wide get harnesses`. Exit `0`. Header includes `CONFIG`. Codex CONFIG contains `.codex/config.toml` under the isolated `homes/home-a`, not the repo `testdata/` path.
- **Describe.** Run `verify-hctl drive --out inventory/describe-codex.txt -- describe harness codex`. Exit `0`. Block includes `Name:              codex`, `Also known as:     openai-codex`, `Default Model:     gpt-5.2-codex`, `Secret:            sha256:`, and no `sk-test`.
- **Empty home.** Run `verify-hctl drive --context empty --out inventory/get-harnesses-empty.txt -- get harnesses`. Exit `0`. Stderr or stdout contains `Next: hctl get harnesses -o wide`.
- **Proof.** Keep the `--out` files. Re-read Codex from disk: `grep '^model =' "$HCTL_VERIFY_HOME_A/.codex/config.toml"` is still `model = "gpt-5.2-codex"` (inventory is read-only).

## Gotchas

- `--home` on `drive` is forbidden. It would ignore the `empty` / `box` context homes.
- `-o yaml` is not valid. It exits `2` with `json|wide`; do not treat that as an inventory format.
- `get harness NAME` still prints a table header. Assert the absence of other harness **rows**, not the absence of the word `NAME`.
- `--json` and `-o json` are equivalent. Proving one JSON entry point is not a skip of the other if the map listed both; this file lists `-o json` as the handle.
- Probe output (VERSION / INSTALLED) depends on the machine PATH. Doctor uses `--no-probe`. Do not assert `INSTALLED=yes` unless `--probe` was requested.
- `cursor-agent` is the inventory name. `cursor` alone is not a harness id.
