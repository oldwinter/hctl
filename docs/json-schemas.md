# JSON schemas (stable fields)

`--json` / `-o json` output is redacted (no `sk-` tokens). Field names below are the 1.0 contract.

## Snapshot (`get harnesses`, `get harness NAME`, `describe harness`)

| field | type | notes |
| --- | --- | --- |
| name | string | official harness id |
| nameAliases | string[] | official aliases (`openai-codex`, `claude-code`, …) |
| installed | bool | binary on PATH (local) or remote `command -v` (SSH) |
| installedPath | string | |
| version | string | cheap `--version` probe (local only; skipped over SSH and with `--no-probe`) |
| configPaths | string[] | |
| configFound | bool | |
| provider | string | |
| baseUrlHost | string | host only, never userinfo |
| defaultModel | string | |
| effort | string | |
| aliases | object | claude role models |
| secretFingerprint | string | sha256 first 8 hex |
| secretPresent | bool | |
| secretRef | string | env var name |
| notes | string[] | |
| parseError | string | |

## DoctorCheck (`doctor`)

`name`, `installed`, `config`, `key`, `onboarding`, `drift`, `message`.

`installed`/`config`/`key`/`onboarding` are `ok|missing|error|needed|env-ref`.
`drift` is empty or `key-drift`.

## ApplyReport (`set`, `apply`, `sync`)

`dryRun`, `changes[]` (`harness`, `field`, `from`, `to`, `path`), `backups[]`, `verified`, optional `secrets[]` and `notes[]`.

`secrets[]`: `harness`, `fromFingerprint`, `toFingerprint`, `secretRef`, `copied`, `action`, `backups[]`. Text `sync` prints secret action lines on stdout with the apply report (not stderr).

## DesiredFile (`apply -f`, `diff -f`)

```toml
apiVersion = "harnessctl/v1"
kind = "DesiredState"
[harnesses.codex]
model = "o4-mini"
provider = "custom"
secretRef = "OPENAI_API_KEY"
```
