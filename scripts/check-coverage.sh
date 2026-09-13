#!/usr/bin/env bash
# Fail if go tool cover total is below the statement-coverage floor.
# Raise MIN when coverage meaningfully increases. Do not chase 100%.
set -euo pipefail

# Current main total rounded down to one decimal (measured 2026-09-13: 58.3%).
MIN="${COVERAGE_MIN:-58.0}"
profile="${1:-coverage.out}"

if [[ ! -f "${profile}" ]]; then
	echo "missing coverprofile: ${profile}" >&2
	exit 1
fi

total="$(go tool cover -func="${profile}" | awk '/^total:/ { print $3 }')"
if [[ -z "${total}" ]]; then
	echo "could not read total coverage from ${profile}" >&2
	exit 1
fi

got="${total%\%}"
if ! awk -v got="${got}" -v min="${MIN}" 'BEGIN { exit !(got+0 >= min+0) }'; then
	echo "coverage ${total} is below floor ${MIN}%" >&2
	exit 1
fi

echo "coverage ${total} (floor ${MIN}%)"
