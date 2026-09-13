#!/usr/bin/env bash
# Cloud Agent environment bootstrap for hctl.
# Idempotent: safe to re-run. Does not start long-running services.
set -euo pipefail

if [ -n "${BASH_SOURCE[0]:-}" ] && [ -f "${BASH_SOURCE[0]}" ]; then
  cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
fi

export PATH="/usr/local/bin:${PATH}"

install_bin_dir=/usr/local/bin
maybe_sudo=()
if [ ! -w "$install_bin_dir" ]; then
  maybe_sudo=(sudo)
fi

echo "==> Go toolchain"
if ! command -v go >/dev/null 2>&1; then
  echo "go is required (see go.mod)" >&2
  exit 1
fi
go version

echo
echo "==> Installing just 1.58.0"
if command -v just >/dev/null 2>&1 && just --version 2>/dev/null | grep -q '1\.58\.0'; then
  echo "$(just --version) already installed"
else
  curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh \
    | "${maybe_sudo[@]}" bash -s -- --tag 1.58.0 --to "$install_bin_dir"
fi

echo
echo "==> Installing golangci-lint v2.13.2"
if command -v golangci-lint >/dev/null 2>&1 && golangci-lint version 2>/dev/null | grep -q 'version 2\.13\.2'; then
  golangci-lint version | head -n 1
else
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
    | "${maybe_sudo[@]}" sh -s -- -b "$install_bin_dir" v2.13.2
fi

echo
echo "==> Downloading Go modules"
go mod download

echo
echo "==> Building binaries (bin/hctl, bin/harnessctl)"
just build

echo
echo "==> Environment ready"
