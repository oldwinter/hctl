# CI

The GitHub Actions workflow lives at `.github/workflows/ci.yml`.

It runs on push/PR:

- `gofmt` (fail if any file needs formatting)
- `go vet ./...`
- `go test -race ./...`
- statement coverage must be **100.0%** (`go test ./... -coverprofile=...` + `go tool cover -func`)
- build `hctl` and `harnessctl`

Locally: `just cover` (same 100% gate).


## Workflow file (copy to `.github/workflows/ci.yml`)

```yaml
name: ci

on:
  push:
    branches: [main, "chore/**"]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22.x"
      - name: gofmt
        run: test -z "$(gofmt -l .)"
      - name: go vet
        run: go vet ./...
      - name: test race
        run: go test -race ./...
      - name: coverage 100%
        run: |
          go test ./... -coverprofile=coverage.out -covermode=atomic
          total="$(go tool cover -func=coverage.out | awk '/^total:/ {print $3}')"
          echo "coverage total: ${total}"
          if [ "${total}" != "100.0%" ]; then
            go tool cover -func=coverage.out | grep -v '100.0%' || true
            echo "coverage must be 100.0%, got ${total}" >&2
            exit 1
          fi
      - name: build
        run: |
          go build -o /tmp/hctl ./cmd/hctl
          go build -o /tmp/harnessctl ./cmd/harnessctl
```
