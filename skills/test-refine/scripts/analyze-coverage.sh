#!/usr/bin/env bash
# Standalone coverage helper: produces a per-function coverage report
# for a given package.
set -euo pipefail

usage() {
    cat <<EOF
Usage: $0 [--pkg <path>] [--output <file>]

Runs 'go test -cover -covermode=atomic' on the package and produces
a per-function coverage breakdown, sorted by uncovered fraction.
EOF
}

PKG="."
OUTPUT=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --pkg) PKG="$2"; shift 2 ;;
        --output) OUTPUT="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
    esac
done

cov_profile="$(mktemp -t test-refine-cov.XXXXXX)"
trap 'rm -f "$cov_profile"' EXIT

go test -cover -covermode=atomic -coverprofile="$cov_profile" "$PKG"

if [[ -n "$OUTPUT" ]]; then
    go tool cover -func="$cov_profile" > "$OUTPUT"
    # Also append a sorted view of least-covered functions.
    {
        echo
        echo "# Least-covered functions (top 20):"
        go tool cover -func="$cov_profile" | grep -v '^total:' | \
            awk '{print $NF, $0}' | sort -nr | tail -20
    } >> "$OUTPUT"
    echo "Wrote $OUTPUT"
else
    go tool cover -func="$cov_profile"
fi
