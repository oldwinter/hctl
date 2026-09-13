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

`just lint` runs [golangci-lint](https://golangci-lint.run/) **v2.13.2** with `.golangci.yml` (`version: "2"`). Install that version; do not fall back to `go vet`:

```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b "$(go env GOPATH)/bin" v2.13.2
```

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

Module path is `github.com/oldwinter/hctl` (https://github.com/oldwinter/hctl). Do **not** push this Go project to the Rust repo `oldwinter/harnessctl`.

`just build` / `just release` inject version metadata via ldflags; plain `go install` leaves commit/date as `unknown`.
