#!/usr/bin/env bash
# Drive hctl the way a new user does. Fail if first-run help or usage drifts.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
bin="${HCTL_BIN:-}"
if [[ -z "$bin" ]]; then
	bin="$(mktemp)"
	go build -o "$bin" "$root/cmd/hctl"
	trap 'rm -f "$bin"' EXIT
fi

fail() { echo "FAIL: $*" >&2; exit 1; }

help_out="$("$bin" --help)"
echo "$help_out" | grep -q 'hctl get harnesses' || fail "root help missing get example"
echo "$help_out" | grep -q 'hctl doctor' || fail "root help missing doctor example"

ver="$("$bin" version)"
echo "$ver" | grep -q 'hctl version' || fail "version line missing"
if echo "$ver" | grep -q 'commit: unknown\|built:  unknown'; then
	echo "$ver" | grep -q 'just build injects commit and date' || fail "unknown metadata missing just-build hint"
fi

set +e
apply_err="$("$bin" apply 2>&1)"
apply_code=$?
set -e
[[ "$apply_code" -eq 2 ]] || fail "apply missing -f exit=$apply_code want 2"
echo "$apply_err" | grep -q 'hctl apply -f' || fail "apply missing-file error has no example"

set +e
desc_err="$("$bin" describe 2>&1)"
desc_code=$?
set -e
[[ "$desc_code" -eq 2 ]] || fail "describe missing args exit=$desc_code want 2"
echo "$desc_err" | grep -q 'hctl describe harness' || fail "describe missing-args error has no example"

set +e
sync_err="$("$bin" sync 2>&1)"
sync_code=$?
set -e
[[ "$sync_code" -eq 2 ]] || fail "sync missing --from/--to exit=$sync_code want 2"
echo "$sync_err" | grep -q 'hctl sync --from' || fail "sync missing-flag error has no example"

set +e
use_err="$("$bin" config use-context 2>&1)"
use_code=$?
set -e
[[ "$use_code" -eq 2 ]] || fail "use-context missing NAME exit=$use_code want 2"
echo "$use_err" | grep -q 'hctl config get-contexts' || fail "use-context missing-args error has no get-contexts hint"

home="$root/testdata/home-a"
cfg="$root/testdata/harnessctl.yaml"
list_out="$("$bin" --home "$home" --config "$cfg" --no-probe list harnesses)"
echo "$list_out" | grep -q codex || fail "list harnesses did not show codex"

ctx_out="$("$bin" --config "$cfg" config current-context)"
echo "$ctx_out" | grep -q mba || fail "current-context missing name"
echo "$ctx_out" | grep -q "$cfg" || fail "current-context missing config path"

echo "verify-first-run: ok"
