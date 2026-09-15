# Doctor

Doctor lets a user see whether each harness is installed, has a readable config, has a key or env-ref, needs onboarding, and whether two harnesses on the same host have drifted keys.

## Sub-features

- `doctor-mba` reports config/key rows for the seeded home-a fixtures.
- `doctor-theme` marks Claude onboarding needed when `~/.claude.json` is incomplete and settings are theme-only.
- `doctor-drift` reports `key drift on host sub2api.example (claude,codex)` for the drift fixture.
- `doctor-json` emits the same checks as JSON.
- `doctor-parse-next` prints `Next: hctl describe harness NAME` when a row's CONFIG is `error` (exit `5`). Skip unless a parse-error fixture is in the workspace.
- `doctor-onboard-inspect` puts `Inspect: hctl describe harness NAME` in MESSAGE when ONBOARDING is `needed` and there is no parse error (missing config or no default model).

## How to get to it (user POV)

- Run `hctl doctor` on the current context.
- Run `hctl --context theme doctor` (or `config use-context theme` then `doctor`).
- Run `hctl --context drift doctor`.
- Run `hctl doctor -o json`.

## Driving it with verify-hctl

Preconditions:

- `verify-hctl doctor` (the helper) reports ready. That check is not a substitute for this product command.
- `mba` is home-a: Codex and Claude configs exist with `sk-test-aaa` on disk (must not appear in output).
- `theme` is the home-theme copy: `.claude.json` lacks `theme` / `hasCompletedOnboarding`; `.claude/settings.json` is theme-only.
- `drift` is the home-drift copy: Codex and Claude share host `sub2api.example` with different fixture keys.

- **Healthy fixture table.** Run `.cursor/skills/verify-hctl/scripts/verify-hctl drive --out doctor/mba.txt -- doctor`. Exit `0`. Header is `NAME INSTALLED CONFIG KEY ONBOARDING DRIFT MESSAGE`. Codex `CONFIG` is `ok`. Claude `CONFIG` is `ok`. No `sk-test` token. OpenCode MESSAGE may mention JSONC comment drop.
- **JSON.** Run `verify-hctl drive --out doctor/mba.json.txt -- -o json doctor`. Exit `0`. JSON includes `"name": "codex"` and no fixture keys.
- **Theme leftover.** Run `verify-hctl drive --context theme --out doctor/theme.txt -- doctor`. Exit `0`. Claude `ONBOARDING` is `needed`. MESSAGE mentions onboarding or wizard leftover / missing theme and `Inspect: hctl describe harness claude`. Settings-only theme without a model is part of that leftover.
- **Key drift.** Run `verify-hctl drive --context drift --out doctor/drift.txt -- doctor`. Exit `0`. DRIFT is `key-drift` for Codex and Claude. MESSAGE contains `key drift on host sub2api.example (claude,codex)`.
- **Proof.** Keep the four `--out` files. Confirm `HCTL_VERIFY_HOME_A/.codex/config.toml` still has `model = "gpt-5.2-codex"` (doctor does not write).

## Gotchas

- Helper `verify-hctl doctor` and product `hctl doctor` are different commands. The helper answers "is this instance worth driving?". The product answers "are these harness configs healthy?".
- INSTALLED is `missing` on most CI machines. That is not a failure for config/key/onboarding proof.
- Drift grouping is by host, not by key value. Both Codex and Claude rows carry the same host message.
- Do not dump fixture files into artifacts; they contain `sk-test-*`. Quote only `model =` / onboarding notes / doctor MESSAGE text.
- `doctor` parse errors exit `5` and print `Next: hctl describe harness NAME`. The shipped fixtures do not include a broken file; do not claim that path verified unless you added a disposable broken copy in the workspace.
