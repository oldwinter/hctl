# Weekly verify-hctl maintenance

You are a scheduled Cloud Agent on `github.com/oldwinter/hctl` (`main`).

Read and follow the pstack skill `maintain-verification-skill` if it is available. If it is not on disk, follow the procedure below. Do not invent a second verification skill.

## Target

`.cursor/skills/verify-hctl/` — `SKILL.md`, `features/`, and `scripts/verify-hctl`.

## Outcomes (pick one)

- **clean** — every feature got source and live coverage; nothing worth shipping. No branch, no PR.
- **changed** — one PR of proven doc, harness, or map corrections, confined to `.cursor/skills/verify-hctl/`.
- **blocked** — coverage could not finish or a proven fix could not ship safely. Say exactly what blocked it.

## Edit scope

Only edit the verification skill directory. Never edit product code. A behavior the map describes that the app no longer does is either doc drift (fix the map) or a product regression (report it, do not paper over it).

## Pass

0. Locate `.cursor/skills/verify-hctl/`. If it is missing, stop and report `blocked`: run `/create-verification-skill` first.
1. Index hygiene: read `features/README.md` and glob sibling files. Fix missing, extra, duplicate, or dead entries.
2. Source wave: one read-only subagent per feature file, in parallel. Each returns summary / source entry points / likely drift or none / one live recipe. Children never drive the app and never edit files.
3. Reconcile. Sweep recent commits for user-facing surfaces missing from the map — require a concrete source path before calling one missing.
4. Live pass: follow `SKILL.md` Launch / Doctor / Drive / Evidence / Cleanup. Exercise every mapped feature at least once. Doctor before the first drive and after any failed drive. Keep evidence under `.cursor/skills/verify-hctl/artifacts/`. Tear down the workspace after the last drive. Evidence stays.
5. Triage: wrong user-POV docs → fix the map. Harness cannot drive working behavior → fix the helper and re-drive live. Broken product behavior → report, keep out of this PR.
6. Ship or stop. For **changed**: one PR, re-read every changed file first. For **clean** or **blocked**: no PR.

## Quality bar

- Do not open a PR for wording-only churn.
- Do not commit scratch run notes.
- Never print plaintext API keys. Fixture keys are `sk-test-aaa` / `sk-test-bbb` only; treat a leak as a failed proof.
- Automations use the model's maximum context window. Prefer the cheapest capable model the automation settings allow.
