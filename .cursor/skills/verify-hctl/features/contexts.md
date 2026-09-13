# Contexts

Contexts let a user see which environment (`mba`, `box`, …) `hctl` will read, switch the current one, and add a local or SSH target — without touching harness dotfiles.

## Sub-features

- `ctx-list` lists configured contexts with a `*` on the current one.
- `ctx-current` prints the current name and the resolved config path.
- `ctx-current-json` includes `currentContext` and `config`.
- `ctx-use` writes `current-context` in the verification config file.
- `ctx-set-local` creates a local context and tells the user to `use-context` when it is not current.
- `ctx-missing-args` exits `2` with an example when `use-context` or `set-context` has no NAME.

## How to get to it (user POV)

- Run `hctl config get-contexts`.
- Run `hctl config current-context` (optionally `-o json`).
- Run `hctl config use-context box`.
- Run `hctl config set-context lab --kind local --home /path`.
- Run `hctl config set-context box --kind ssh --ssh user@host` (usage example only here: verification `box` is already local).

## Driving it with verify-hctl

Preconditions:

- `verify-hctl doctor` reports ready.
- Verification config already has `mba`, `box`, `theme`, `drift`, `empty`. Current context is `mba`.
- Do not point `--home` at a new context; `set-context --home` writes a path into the **config file**, it does not need a populated tree unless you later `get` against it.

- **List.** Run `.cursor/skills/verify-hctl/scripts/verify-hctl drive --out contexts/get-contexts.txt -- config get-contexts`. Exit `0`. Header is `CURRENT NAME KIND HOME SSH`. `mba` has `*` and `KIND` `local`. `box` is `local` (this is the isolated copy, not SSH).
- **Current.** Run `verify-hctl drive --out contexts/current.txt -- config current-context`. Exit `0`. First line is `mba`. Second line is the workspace `config.yaml` path (from `verify-hctl status`).
- **Current JSON.** Run `verify-hctl drive --out contexts/current.json.txt -- -o json config current-context`. Exit `0`. JSON has `"currentContext": "mba"` and the same config path.
- **Switch.** Run `verify-hctl drive --out contexts/use-box.txt -- config use-context box`. Exit `0`. Stdout is `Switched to context "box".`. Then `drive -- config current-context` prints `box`. Then `drive -- get harness codex` shows MODEL `o4-mini` (home-b).
- **Restore mba.** Run `verify-hctl drive -- config use-context mba` so later features start from baseline.
- **Set local.** Run `verify-hctl drive --out contexts/set-lab.txt -- config set-context lab --kind local --home /tmp/hctl-verify-unused-lab`. Exit `0`. Stdout contains `Context "lab" saved.` and `Next: hctl config use-context lab`. `get-contexts` now includes `lab`.
- **Missing NAME.** Run `verify-hctl drive --out contexts/use-missing.txt -- config use-context` ; expect exit `2` and text naming `hctl config get-contexts`. Run `verify-hctl drive --out contexts/set-missing.txt -- config set-context` ; expect exit `2` and an ssh example (`hctl config set-context box --kind ssh`).
- **Proof.** `config.yaml` in the workspace (not `~/.hctl/config.yaml`) contains `current-context: mba` after restore. Artifacts remain under `artifacts/contexts/`.

## Gotchas

- Verification `box` is `kind: local` on purpose so `sync`/`diff --contexts` do not need SSH. Do not "fix" it back to the repo's `testdata/harnessctl.yaml` SSH stub.
- `config use-context` writes the resolved `--config` file. If you omit the helper, you can overwrite the user's `~/.hctl/config.yaml`.
- `set-context` does not switch current-context. A missing `Next: hctl config use-context` on a newly added name is a bug.
- `config get-contexts` on a missing default file prints `Next: config file not created yet; …`. That path is not this launch (the workspace file exists).
- Do not run real `set-context --kind ssh` against a live host in this skill.
