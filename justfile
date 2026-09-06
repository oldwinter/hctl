# harnessctl developer tasks

binary := "bin/harnessctl"
alias_bin := "bin/hctl"
version := "1.0.0"
commit := `git rev-parse --short HEAD 2>/dev/null || echo unknown`
date := `date -u +%Y-%m-%dT%H:%M:%SZ`

ldflags := "-X github.com/oldwinter/harnessctl/internal/cli.Version=" + version + " -X github.com/oldwinter/harnessctl/internal/cli.Commit=" + commit + " -X github.com/oldwinter/harnessctl/internal/cli.Date=" + date

build:
	mkdir -p bin
	go build -ldflags="{{ldflags}}" -o {{binary}} ./cmd/harnessctl
	ln -sfn harnessctl {{alias_bin}}

test:
	go test ./...

race:
	go test -race ./internal/mutate ./internal/fsx ./internal/cli ./internal/edit

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
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml -o wide get harnesses
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml --json get models
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml describe harness codex
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml doctor
	{{binary}} --config testdata/harnessctl.yaml diff harness codex --home-a testdata/home-a --home-b testdata/home-b
	{{binary}} --config testdata/harnessctl.yaml config get-contexts
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml set model codex o4-mini --dry-run
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml diff -f testdata/desired.toml
	{{binary}} --home testdata/home-a --config testdata/harnessctl.yaml apply -f testdata/desired.toml --dry-run
	{{binary}} completion bash >/dev/null
