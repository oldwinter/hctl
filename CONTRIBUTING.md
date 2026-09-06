# Contributing

## Dev loop

```bash
just test
just race
just lint
just smoke
just fmt
just release
```

`go test ./...` must stay green. Fixtures live under `testdata/` and use only fake keys (`sk-test-aaa`, `sk-test-bbb`).

## Layout

- `cmd/hctl`, `cmd/harnessctl` — binaries (`hctl` is primary)
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
- Do not publish to `github.com/oldwinter/harnessctl` — that is a different Rust project.

## Release

1. Bump `internal/cli.Version` and `justfile` `version`.
2. Update `CHANGELOG.md`.
3. Run `just test && just race && just smoke && just release`.
4. Tag `v1.x.y` and push.

```bash
git tag v1.0.1
git push origin v1.0.1
```

Module path is `github.com/oldwinter/hctl`. After the GitHub remote exists:

```bash
git remote add github git@github.com:oldwinter/hctl.git
git push github main --tags
```

If `gh` can create the empty repo:

```bash
gh repo create oldwinter/hctl --public --source=. --remote=github --push
git tag v1.0.1
git push github v1.0.1
```
