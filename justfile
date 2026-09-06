# harnessctl developer tasks
# Usage: just build | just test | just fmt | just lint | just smoke

binary := "bin/harnessctl"
alias_bin := "bin/hctl"

build:
	mkdir -p bin
	go build -ldflags="-X github.com/oldwinter/harnessctl/internal/cli.Version=0.1.0" -o {{binary}} ./cmd/harnessctl
	ln -sfn harnessctl {{alias_bin}}

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	#!/usr/bin/env bash
	set -euo pipefail
	if command -v golangci-lint >/dev/null 2>&1; then
		golangci-lint run ./...
	else
		echo "golangci-lint not installed; falling back to go vet"
		go vet ./...
	fi

smoke: build
	{{binary}} version
	{{alias_bin}} version
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml get harnesses
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml --json get models
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml describe harness codex
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml doctor
	{{binary}} --config testdata/harnessctl.yaml diff harness codex --home-a testdata/home-a --home-b testdata/home-b
	{{binary}} --config testdata/harnessctl.yaml config get-contexts
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml set model codex o4-mini --dry-run
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml diff -f testdata/desired.toml
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml apply -f testdata/desired.toml --dry-run
