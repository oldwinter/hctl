# Contributing

## Dev loop

```bash
just test
just race
just lint
just smoke
just fmt
```

`go test ./...` must stay green. Fixtures live under `testdata/` and use only fake keys (`sk-test-aaa`, `sk-test-bbb`).

## Layout

- `cmd/harnessctl`, `cmd/hctl` — binaries
- `internal/cli` — Cobra
- `internal/adapters/<harness>` — readers/writers
- `internal/edit` — TOML/JSON/YAML/JSONC mutations
- `internal/fsx` — local + SSH filesystem
- `internal/mutate` — backup + verify
- `internal/remote` — context dial

## Rules

- Do not print or commit real secrets.
- Do not add agent dispatch / Herdr orchestration.
- New adapters need `testdata/` fixtures and a leak test (`sk-test` must not appear in table/JSON/`String()`).
- Prefer boring file IO. Writers must re-read after write.

## Release

1. Bump `internal/cli.Version` and `justfile` `version`.
2. Update `CHANGELOG.md`.
3. Tag `v1.x.y` after `just test && just smoke`.

```bash
git tag v1.0.0
git push origin v1.0.0
```

Module path stays `github.com/oldwinter/harnessctl`. After the GitHub remote exists:

```bash
git remote add github git@github.com:oldwinter/harnessctl.git
git push github main --tags
```
