#!/usr/bin/env bash
# Wrapper around `gremlins unleash` that enforces skill conventions:
# - JSON output to a known location
# - Optional --pkg to scope to a single package
# - Optional --tags for build tags (e.g., integration)
# - Optional --integration to enable integration test runs
# - Optional --silent for CI use
# - Default output: .reviews/mutations/<pkg-slug>.json
set -euo pipefail

usage() {
    cat <<EOF
Usage: $0 [--pkg <path>] [--output <file>] [--tags "<tags>"] [--integration] [--silent] [--config <yaml>] [--dry-run]

  --pkg          Package path to test (default: current directory).
  --output       JSON output file path (default: .reviews/mutations/<slug>.json).
  --tags         Go build tags (e.g., "integration").
  --integration  Run integration tests as well.
  --silent       Silent mode (only errors on stdout).
  --config       Path to gremlins config yaml.
  --dry-run      Show what mutations would run without executing tests.

The wrapper writes JSON output for downstream tools (analyze-survivors.sh,
test-refine skill).
EOF
}

PKG="."
OUTPUT=""
TAGS=""
INTEGRATION=0
SILENT=0
CONFIG=""
DRY_RUN=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --pkg)         PKG="$2"; shift 2 ;;
        --output)      OUTPUT="$2"; shift 2 ;;
        --tags)        TAGS="$2"; shift 2 ;;
        --integration) INTEGRATION=1; shift ;;
        --silent)      SILENT=1; shift ;;
        --config)      CONFIG="$2"; shift 2 ;;
        --dry-run)     DRY_RUN=1; shift ;;
        -h|--help)     usage; exit 0 ;;
        *) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if ! command -v gremlins >/dev/null 2>&1; then
    echo "error: gremlins not installed. Run install-gremlins.sh first." >&2
    exit 1
fi

# Default output path: .reviews/mutations/<slug>.json relative to cwd.
if [[ -z "$OUTPUT" ]]; then
    slug="$(echo "$PKG" | sed 's|[^a-zA-Z0-9]|_|g' | sed 's|^_*||;s|_*$||')"
    [[ -z "$slug" ]] && slug="root"
    OUTPUT=".reviews/mutations/${slug}.json"
fi

mkdir -p "$(dirname "$OUTPUT")"

args=(unleash)
[[ -n "$TAGS"   ]] && args+=(--tags "$TAGS")
[[ "$INTEGRATION" -eq 1 ]] && args+=(--integration)
[[ "$SILENT"      -eq 1 ]] && args+=(--silent)
[[ -n "$CONFIG" ]] && args+=(--config "$CONFIG")
[[ "$DRY_RUN"     -eq 1 ]] && args+=(--dry-run)
args+=(--output "$OUTPUT" "$PKG")

echo "Running: gremlins ${args[*]}"
gremlins "${args[@]}"
echo "JSON results: $OUTPUT"
