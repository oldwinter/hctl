# Set a field

Set lets a user change one harness field (`model` or `provider`) with a dry-run preview, then an atomic write that is re-read and verified.

## Sub-features

- `set-model-dry-run` shows the pending Codex model change and writes nothing.
- `set-model-write` writes Codex `model`, prints `verified: ok`, and leaves a backup.
- `set-model-reread` shows the new model from `describe` and from the file.
- `set-provider-claude` refuses Claude provider (usage) even with `--dry-run`.
- `set-provider-grok` refuses Grok provider (usage) even with `--dry-run`.

## How to get to it (user POV)

- Run `hctl set model codex o4-mini --dry-run`.
- Run `hctl set model codex o4-mini`.
- Run `hctl describe harness codex` (or `get models`) to confirm.
- Run `hctl set provider claude custom` or `hctl set provider grok custom` (expected refusal).

## Driving it with verify-hctl

Preconditions:

- `verify-hctl doctor` reports ready.
- Current context is `mba`. Codex file is the home-a copy: `model = "gpt-5.2-codex"`.
- `$HCTL_VERIFY_BACKUP` is empty of Codex backups at the start of the dry-run (or record its listing first).
- No ownership pointer under the copied home (`homes/home-a/.config/harness/ownership.json` must not exist unless you are proving the managed-path refusal; this file does not).

- **Dry-run.** Read `grep '^model =' "$HCTL_VERIFY_HOME_A/.codex/config.toml"` (`gpt-5.2-codex`). Run `.cursor/skills/verify-hctl/scripts/verify-hctl drive --out set-model/dry-run.txt -- set model codex o4-mini --dry-run`. Exit `0`. Stdout contains `dry-run: no files written`, FIELD `model` with TO `o4-mini`, and PATH ending in `.codex/config.toml` under the isolated home. Re-read the file: still `model = "gpt-5.2-codex"`. Backup dir has no new Codex backup.
- **Write.** Run `verify-hctl drive --out set-model/write.txt -- set model codex o4-mini`. Exit `0`. Stdout contains `verified: ok` and a `backup:` line under `$HCTL_VERIFY_BACKUP`.
- **Reread CLI.** Run `verify-hctl drive --out set-model/describe.txt -- describe harness codex`. Exit `0`. `Default Model:` is `o4-mini`.
- **Reread file.** `grep '^model =' "$HCTL_VERIFY_HOME_A/.codex/config.toml"` is `model = "o4-mini"`. Unrelated keys (`model_provider`, `experimental_bearer_token`) remain. Do not copy the bearer line into artifacts.
- **Refuse Claude provider.** Run `verify-hctl drive --out set-model/claude-provider.txt -- set provider claude custom --dry-run`. Exit `2`. Text contains `unsupported`. File still has Claude model `claude-sonnet-4`.
- **Refuse Grok provider.** Run `verify-hctl drive --out set-model/grok-provider.txt -- set provider grok custom --dry-run`. Exit `2`. Text contains `unsupported`.
- **Proof.** Keep the `--out` files, the `describe` reread, the `model =` grep, and the backup path from the write step.

## Gotchas

- Already-converged `set` (Codex already `o4-mini`) skips backup and write. Prove dry-run/write from `gpt-5.2-codex`, or reset the file from `testdata/home-a/.codex/config.toml` first.
- Claude, Grok, and Factory Droid `set provider` must fail on `--dry-run` too. A silent `no changes` is a bug.
- Hermes/OpenCode/pi/droid have their own write rules (Hermes inline key refuse, droid provider unsupported). This feature file does not cover them; do not claim they were verified here.
- Ownership manifests fail closed. If an agent creates `~/.config/harness/ownership.json` inside the copy, `set` will refuse. That is a different path.
- JSONC comments are dropped on write for OpenCode. Codex is TOML; comments on unrelated keys stay.
- Never pass `--fields secret` or print the token. Fingerprint-only is the only allowed secret evidence.
