# hctl developer tasks

binary := "bin/hctl"
alias_bin := "bin/harnessctl"
version := "1.0.1"
commit := `git rev-parse --short HEAD 2>/dev/null || echo unknown`
date := `date -u +%Y-%m-%dT%H:%M:%SZ`

ldflags := "-X github.com/oldwinter/hctl/internal/cli.Version=" + version + " -X github.com/oldwinter/hctl/internal/cli.Commit=" + commit + " -X github.com/oldwinter/hctl/internal/cli.Date=" + date

build:
	mkdir -p bin
	go build -ldflags="{{ldflags}}" -o {{binary}} ./cmd/hctl
	go build -ldflags="{{ldflags}}" -o {{alias_bin}} ./cmd/harnessctl

test:
	go test ./...

race:
	go test -race ./internal/mutate ./internal/fsx ./internal/cli ./internal/edit

fmt:
	go fmt ./...

# golangci-lint v2.13.2; config is .golangci.yml (version: "2").
lint:
	#!/usr/bin/env bash
	set -euo pipefail
	if ! command -v golangci-lint >/dev/null 2>&1; then
		echo "golangci-lint v2.13.2 is required (see CONTRIBUTING.md)" >&2
		exit 1
	fi
	golangci-lint run ./...

# Cross-compile linux/amd64 binaries and SHA-256 checksums into dist/.
release:
	#!/usr/bin/env bash
	set -euo pipefail
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build -ldflags="{{ldflags}}" -o dist/hctl_linux_amd64 ./cmd/hctl
	GOOS=linux GOARCH=amd64 go build -ldflags="{{ldflags}}" -o dist/harnessctl_linux_amd64 ./cmd/harnessctl
	( cd dist && sha256sum hctl_linux_amd64 harnessctl_linux_amd64 > SHA256SUMS )
	echo "checksums:"
	cat dist/SHA256SUMS

smoke: build
	{{binary}} version
	{{alias_bin}} version
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml get harnesses
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml -o wide get harnesses
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml --json get models
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml describe harness codex
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml doctor
	{{binary}} --config testdata/harnessctl.yaml diff harness codex --home-a testdata/home-a --home-b testdata/home-b
	{{binary}} --config testdata/harnessctl.yaml config get-contexts
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml set model codex o4-mini --dry-run
	{{binary}} --config testdata/harnessctl.yaml diff -f testdata/desired.toml
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml apply -f testdata/desired.toml --dry-run
	{{binary}} completion bash >/dev/null
