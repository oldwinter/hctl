#!/usr/bin/env bash
# Idempotent bootstrap for the hctl / harnessctl Go control-plane.
# Installs dev tooling not present in the base image, then downloads
# modules and builds the binaries so the workspace is ready to use.
set -euo pipefail

JUST_VERSION="1.58.0"
GOLANGCI_VERSION="v1.61.0"

log() { printf '\n==> %s\n' "$1"; }

log "Go toolchain"
go version

if ! command -v just >/dev/null 2>&1; then
  log "Installing just ${JUST_VERSION}"
  curl --proto '=https' --tlsv1.2 -sSfL https://just.systems/install.sh \
    | sudo bash -s -- --tag "${JUST_VERSION}" --to /usr/local/bin
else
  log "just already installed: $(just --version)"
fi

if ! command -v golangci-lint >/dev/null 2>&1; then
  log "Installing golangci-lint ${GOLANGCI_VERSION}"
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
    | sudo sh -s -- -b /usr/local/bin "${GOLANGCI_VERSION}"
else
  log "golangci-lint already installed: $(golangci-lint --version)"
fi

log "Downloading Go modules"
go mod download

log "Building binaries (bin/hctl, bin/harnessctl)"
just build

log "Environment ready"
