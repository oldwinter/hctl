# CI

The GitHub Actions workflow lives at `.github/workflows/ci.yml`.

It runs on push/PR:

- `gofmt` (fail if any file needs formatting)
- `go vet ./...`
- `go test -race ./...`
- statement coverage must be **100.0%** (`go test ./... -coverprofile=...` + `go tool cover -func`)
- build `hctl` and `harnessctl`

Locally: `just cover` (same 100% gate).
