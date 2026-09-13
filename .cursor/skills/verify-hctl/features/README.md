# hctl verification map

This directory is the maintained source for verifying the user-facing behavior of `hctl` (secondary binary `harnessctl`). Read the index before driving the app, then use the matching feature file as the recipe.

## Baseline preconditions

- Launch with `.cursor/skills/verify-hctl/scripts/verify-hctl launch` and keep that `RUN_ID`.
- Run `.cursor/skills/verify-hctl/scripts/verify-hctl doctor` and require `doctor: ready`.
- Drive only through `verify-hctl drive`. Never pass `--home`. Never export `HCTL_HOME` to the real user home.
- `HCTL_SSH=0` is set by the helper. Do not unset it.
- Seeded homes are copies: `mba` is home-a (Codex `gpt-5.2-codex`), `box` is home-b (Codex `o4-mini`), plus `theme`, `drift`, and `empty`.
- Put proof files under `.cursor/skills/verify-hctl/artifacts/<feature>/`.
- Never drive an instance that this run did not launch.

## Driving conventions

- Start every recipe from the baseline launch state unless the feature says to mutate first.
- Treat every command as literal. Keep harness names, context names, and flags unchanged.
- Default context is `mba`. Switch with `drive --context NAME` or `config use-context`.
- After a mutation, restore only the copied home (re-launch or recopy that tree). Do not restore by deleting artifacts.
- `list` is an alias of `get`. Prove both entry points when the feature lists them.

## Proof and skip reporting

- Capture the user action and the resulting state, not only the final table.
- CLI proof includes the command, stdout, stderr, and exit code (`drive --out`).
- Mutation proof includes a second user-facing read **and** the on-disk field.
- Dry-run proof includes an unchanged file and no new backup.
- Record the feature ID and entry point used with every artifact.
- Report an unreachable path with the attempted command and the unmet precondition.
- Do not report a skipped entry point as verified through a different path.
- A leaked `sk-test-aaa` / `sk-test-bbb` is a failed proof, not a warning.

## Feature entry contract

Each feature file starts with an H1 title and one paragraph describing the user-visible behavior. It then uses exactly four H2 sections in this order.

1. `Sub-features` lists short IDs with one line for each behavior.
2. `How to get to it (user POV)` lists every user entry point.
3. `Driving it with verify-hctl` starts with `Preconditions:` and uses labeled bullets that pair each user action with an exact command and observable result.
4. `Gotchas` lists traps that can waste or invalidate a verification run.

Keep implementation details out of the map. Name only user paths, stable handles, required state, commands, and observable proof.

## Features

- [Inventory](./inventory.md) covers `get`/`list` harnesses and models, one-name select, `-o json`/`-o wide`, `describe`, and the empty-home next-step hint.
- [Doctor](./doctor.md) covers install/config/key/onboarding rows, Claude theme leftover, and same-host key-drift.
- [Contexts](./contexts.md) covers listing, current-context, `use-context`, and `set-context`.
- [Set a field](./set-model.md) covers dry-run and write for `set model`, plus refused `set provider` on Claude/Grok.
- [Apply, diff, and sync](./apply-diff-sync.md) covers desired-state diff/apply and copying a model from `mba` to `box`.
