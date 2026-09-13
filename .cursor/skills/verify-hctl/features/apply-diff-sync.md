# Apply, diff, and sync

Apply writes a desired-state file onto the current context. Diff compares two homes or desired vs current. Sync copies selected fields from one context to another.

## Sub-features

- `diff-homes` shows Codex `defaultModel` (and related fields) between `mba` (gpt-5.2-codex) and `box` (o4-mini).
- `diff-desired` shows pending desired.toml changes against home-a, including the Codex config PATH.
- `apply-dry-run` previews desired.toml and leaves files unchanged.
- `apply-write` applies desired.toml and `get models` shows Codex `o4-mini` and Claude `claude-opus-4`.
- `sync-dry-run` previews copying Codex `model` from `mba` to `box` without writing.
- `sync-write` copies Codex `model` from `mba` to `box` and box Codex file matches.

## How to get to it (user POV)

- Run `hctl diff harness codex --contexts mba,box`.
- Run `hctl diff harness codex --home-a <a> --home-b <b>` (fixture flags; the helper already encodes homes in contexts — prefer `--contexts`).
- Run `hctl diff -f desired.toml`.
- Run `hctl apply -f desired.toml --dry-run` then `hctl apply -f desired.toml`.
- Run `hctl sync --from mba --to box --harness codex --fields model --dry-run` then without `--dry-run`.

## Driving it with verify-hctl

Preconditions:

- `verify-hctl doctor` reports ready.
- Current context is `mba`. Codex on mba is still `gpt-5.2-codex` unless a prior feature mutated it; if it did, recopy `testdata/home-a` onto `$HCTL_VERIFY_HOME_A` or re-launch.
- Box Codex is `o4-mini`.
- Desired file is `$HCTL_VERIFY_DESIRED` (copy of `testdata/desired.toml`): Codex `o4-mini`, Claude `claude-opus-4`, Grok `grok-4.5`, Hermes `anthropic/claude-sonnet-4`, OpenCode `acme/gpt-4.1`.
- Do not pass `--home` on `sync` or `diff --contexts`.

- **Diff contexts.** Run `.cursor/skills/verify-hctl/scripts/verify-hctl drive --out apply-diff-sync/diff-contexts.txt -- diff harness codex --contexts mba,box`. Exit `0`. Output has `FIELD` `defaultModel` (or `model`) with A `gpt-5.2-codex` and B `o4-mini`. No `sk-test`.
- **Diff desired.** Run `verify-hctl drive --out apply-diff-sync/diff-desired.txt -- diff -f "$HCTL_VERIFY_DESIRED"` (expand the path from `status`). Exit `0`. Contains `o4-mini` and a PATH under isolated `homes/home-a/.codex/config.toml`.
- **Apply dry-run.** Snapshot `grep '^model =' "$HCTL_VERIFY_HOME_A/.codex/config.toml"`. Run `verify-hctl drive --out apply-diff-sync/apply-dry-run.txt -- apply -f "$HCTL_VERIFY_DESIRED" --dry-run`. Exit `0`. `dry-run: no files written`. File still `gpt-5.2-codex`.
- **Apply write.** Run `verify-hctl drive --out apply-diff-sync/apply-write.txt -- apply -f "$HCTL_VERIFY_DESIRED"`. Exit `0`. `verified: ok` and at least one `backup:` line. Then `verify-hctl drive --out apply-diff-sync/models-after-apply.txt -- get models`. Codex MODEL is `o4-mini`. Claude MODEL is `claude-opus-4`.
- **Reset box if needed.** Box Codex should still be `o4-mini` after apply (apply targets current context `mba` only).
- **Sync dry-run (opposite direction).** First put mba Codex back to `gpt-5.2-codex` only if you need a visible model delta, **or** sync Claude model from mba (now `claude-opus-4`) to box (home-b Claude is `claude-sonnet-4`). Prefer Claude for a clear delta after apply: run `verify-hctl drive --out apply-diff-sync/sync-dry-run.txt -- sync --from mba --to box --harness claude --fields model --dry-run`. Exit `0`. `dry-run: no files written`. Box Claude file still `claude-sonnet-4`.
- **Sync write.** Run `verify-hctl drive --out apply-diff-sync/sync-write.txt -- sync --from mba --to box --harness claude --fields model`. Exit `0`. `verified: ok`. `verify-hctl drive --context box -- describe harness claude` shows `claude-opus-4`. `grep '"model"' "$HCTL_VERIFY_HOME_B/.claude/settings.json"` contains `claude-opus-4`.
- **Proof.** Keep every `--out` file plus the file greps. Confirm artifacts are under `artifacts/apply-diff-sync/` after cleanup.

## Gotchas

- `apply` / `sync` preflight the whole batch. One managed ownership hit blocks every harness in that invocation; no partial writes.
- Default `sync --fields model,provider` skips unsupported provider on Claude, Grok, and droid. Asking for provider-only on Claude is a usage error; omitting Claude from `--harness` is not required for default fields.
- `diff` without `--contexts`, `--home-a/--home-b`, or `--a/--b` exits `2`.
- `sync` without `--from`/`--to` exits `2` and prints a `--dry-run` example.
- `--fields secret` copies bearer bytes without logging them. Do not use it in this map. Prefer `secret-ref`.
- Desired unknown harness names fail the same way as `apply` (`unknown harness`).
- After `apply`, Codex on mba equals box (`o4-mini`). A following Codex `sync --fields model` is a no-op. Use Claude (or reset mba) for a visible sync.
