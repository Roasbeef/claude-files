#!/usr/bin/env bash
# Phase A: triage. Read-only analysis of an existing Go test suite.
# Produces a markdown report under .reviews/test-refinement/.
set -euo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS="$SKILL_DIR/scripts"
BIN_DIR="${TEST_REFINE_BIN_DIR:-$HOME/.claude/cache/test-refine-bin}"

# build_analyzer compiles a single-file Go program into BIN_DIR.
# It's idempotent: if the binary is newer than the source, it skips.
build_analyzer() {
    local src="$1" name; name="$(basename "$src" .go)"
    local out="$BIN_DIR/$name"
    if [[ -x "$out" && "$out" -nt "$src" ]]; then
        echo "$out"; return
    fi
    mkdir -p "$BIN_DIR"
    # Build in a temp module so 'go build' has a context.
    local td; td="$(mktemp -d -t test-refine-build.XXXXXX)"
    cp "$src" "$td/"
    (
        cd "$td"
        cat > go.mod <<EOF
module local

go 1.22
EOF
        go build -o "$out" "./$(basename "$src")"
    )
    rm -rf "$td"
    echo "$out"
}

usage() {
    cat <<EOF
Usage: $0 [--scope file|package|diff|repo] [options]

  --scope MODE       Scope to analyze:
                       package (default) - current dir's package
                       file              - single test file (--target required)
                       diff              - test files changed vs --base
                       repo              - whole module
  --target PATH      Target file (with --scope file).
  --pkg PATH         Package path (with --scope package). Default: '.'.
  --base BRANCH      Base branch (with --scope diff). Default: 'main'.
  --use-mutations    Run gremlins to feed S12 (mutation-survivor) findings.
  --weights SPEC     Override score weights, e.g. risk=0.6,severity=0.3,gap=0.1.
  --no-coverage      Skip coverage analysis (faster, less signal).

Outputs:
  .reviews/test-refinement/<DATE>-<scope-slug>.md          (report)
  .reviews/test-refinement/_findings/<slug>.json           (raw findings)
EOF
}

SCOPE="package"
TARGET=""
PKG="."
BASE="main"
USE_MUTATIONS=0
WEIGHTS="risk=0.5,severity=0.3,gap=0.2"
DO_COVERAGE=1

while [[ $# -gt 0 ]]; do
    case "$1" in
        --scope) SCOPE="$2"; shift 2 ;;
        --target) TARGET="$2"; shift 2 ;;
        --pkg) PKG="$2"; shift 2 ;;
        --base) BASE="$2"; shift 2 ;;
        --use-mutations) USE_MUTATIONS=1; shift ;;
        --weights) WEIGHTS="$2"; shift 2 ;;
        --no-coverage) DO_COVERAGE=0; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
    esac
done

# --- Resolve target test files ---
TEST_FILES=()
case "$SCOPE" in
    file)
        if [[ -z "$TARGET" ]]; then
            echo "error: --scope file requires --target" >&2; exit 2
        fi
        TEST_FILES=("$TARGET")
        ;;
    package)
        while IFS= read -r f; do TEST_FILES+=("$f"); done < <(
            find "$PKG" -maxdepth 1 -type f -name '*_test.go'
        )
        ;;
    diff)
        while IFS= read -r f; do
            [[ -n "$f" && -f "$f" ]] && TEST_FILES+=("$f")
        done < <(git diff --name-only "$BASE...HEAD" -- '*_test.go' || true)
        ;;
    repo)
        while IFS= read -r f; do TEST_FILES+=("$f"); done < <(
            find . -type f -name '*_test.go' -not -path './vendor/*'
        )
        ;;
    *)
        echo "error: invalid --scope: $SCOPE" >&2; exit 2 ;;
esac

if [[ ${#TEST_FILES[@]} -eq 0 ]]; then
    echo "error: no test files found for scope=$SCOPE" >&2
    exit 1
fi

echo "Resolved scope: $SCOPE (${#TEST_FILES[@]} test file(s))"

# --- Output paths ---
DATE="$(date +%Y-%m-%d)"
slug_for_pkg() {
    local p="$1"
    p="${p#./}"
    p="${p%/}"
    if [[ -z "$p" || "$p" == "." ]]; then
        echo "$(basename "$(pwd)")"
    else
        echo "$p" | tr '/' '_'
    fi
}
case "$SCOPE" in
    file)    SLUG="file-$(basename "$TARGET" .go)" ;;
    package) SLUG="package-$(slug_for_pkg "$PKG")" ;;
    diff)    SLUG="diff-$(git rev-parse --abbrev-ref HEAD | tr '/' '_')" ;;
    repo)    SLUG="repo" ;;
esac

REPORT_DIR=".reviews/test-refinement"
FINDINGS_DIR="$REPORT_DIR/_findings"
mkdir -p "$REPORT_DIR" "$FINDINGS_DIR"

REPORT="$REPORT_DIR/$DATE-$SLUG.md"
FINDINGS="$FINDINGS_DIR/$SLUG.json"
COV_OUT="$FINDINGS_DIR/$SLUG-coverage.txt"
GREMLINS_OUT="$FINDINGS_DIR/$SLUG-gremlins.json"

# --- Build-tag preflight ---
# When test files in scope use //go:build tags (typical for itest/, _e2e
# files, etc.), 'go test' without --tags will silently skip the entire
# file from the build. Coverage will then read 0.0% — misleading. Detect
# this case and warn before running coverage.
BUILD_TAGS_FOUND=()
for f in "${TEST_FILES[@]}"; do
    [[ -f "$f" ]] || continue
    # Match `//go:build` directives in the first 30 lines.
    tag_line="$(head -n 30 "$f" 2>/dev/null | grep -m1 '^//go:build ' || true)"
    if [[ -n "$tag_line" ]]; then
        BUILD_TAGS_FOUND+=("$f: $tag_line")
    fi
done
if [[ ${#BUILD_TAGS_FOUND[@]} -gt 0 && "$DO_COVERAGE" -eq 1 ]]; then
    echo "warn: ${#BUILD_TAGS_FOUND[@]} test file(s) in scope use //go:build directives." >&2
    echo "      Coverage will be misleading unless you re-run with --no-coverage" >&2
    echo "      or invoke 'go test' with matching --tags. Affected files:" >&2
    for entry in "${BUILD_TAGS_FOUND[@]}"; do
        echo "      - $entry" >&2
    done
fi

# --- Step 1: coverage ---
if [[ "$DO_COVERAGE" -eq 1 ]]; then
    echo "Running coverage..."
    cov_profile="$(mktemp -t test-refine-cov.XXXXXX)"
    if [[ "$SCOPE" == "package" ]]; then
        go test -cover -covermode=atomic -coverprofile="$cov_profile" "$PKG" >/dev/null 2>&1 || true
    elif [[ "$SCOPE" == "repo" ]]; then
        go test -cover -covermode=atomic -coverprofile="$cov_profile" ./... >/dev/null 2>&1 || true
    else
        # For file/diff scope, coverage of the containing packages.
        pkgs="$(printf '%s\n' "${TEST_FILES[@]}" | xargs -n1 dirname | sort -u | tr '\n' ' ')"
        go test -cover -covermode=atomic -coverprofile="$cov_profile" $pkgs >/dev/null 2>&1 || true
    fi
    if [[ -s "$cov_profile" ]]; then
        go tool cover -func="$cov_profile" > "$COV_OUT" || true
    else
        : > "$COV_OUT"
    fi
fi

# --- Step 2: optional gremlins ---
if [[ "$USE_MUTATIONS" -eq 1 ]]; then
    if command -v gremlins >/dev/null 2>&1; then
        echo "Running gremlins..."
        case "$SCOPE" in
            package)
                "$HOME/.claude/skills/mutation-testing/scripts/unleash.sh" \
                    --pkg "$PKG" --output "$GREMLINS_OUT" --silent || true
                ;;
            file|diff|repo)
                echo "warn: --use-mutations only supported with --scope package; skipping" >&2
                ;;
        esac
    else
        echo "warn: gremlins not installed; skipping mutation analysis" >&2
        echo "      install via ~/.claude/skills/mutation-testing/scripts/install-gremlins.sh" >&2
    fi
fi

# --- Build analyzer binaries (cached) ---
echo "Building analyzers..."
SMELLS_BIN="$(build_analyzer "$SCRIPTS/detect-smells.go")"
DUPES_BIN="$(build_analyzer "$SCRIPTS/detect-duplicates.go")"
DOMAIN_BIN="$(build_analyzer "$SCRIPTS/domain-checks.go")"
SCORE_BIN="$(build_analyzer "$SCRIPTS/score.go")"

# --- Step 3: AST analysis ---
echo "Detecting smells..."
SMELLS_RAW="$(mktemp -t test-refine-smells.XXXXXX)"
"$SMELLS_BIN" "${TEST_FILES[@]}" > "$SMELLS_RAW"

echo "Detecting duplicates..."
DUPES_RAW="$(mktemp -t test-refine-dupes.XXXXXX)"
"$DUPES_BIN" "${TEST_FILES[@]}" > "$DUPES_RAW"

echo "Domain checks..."
DOMAIN_RAW="$(mktemp -t test-refine-domain.XXXXXX)"
case "$SCOPE" in
    package)
        "$DOMAIN_BIN" --pkg "$PKG" > "$DOMAIN_RAW" || true ;;
    file)
        "$DOMAIN_BIN" --pkg "$(dirname "$TARGET")" > "$DOMAIN_RAW" || true ;;
    *)
        : > "$DOMAIN_RAW"
        echo "info: domain checks skipped for scope=$SCOPE" >&2
        ;;
esac

# --- Step 4: cross-reference gremlins survivors with smells ---
if [[ -s "$GREMLINS_OUT" ]] && command -v jq >/dev/null 2>&1; then
    echo "Cross-referencing mutation survivors..."
    S12_RAW="$(mktemp -t test-refine-s12.XXXXXX)"
    jq -r '
      .files[]
      | . as $f
      | .mutations[]
      | select(.status == "LIVED")
      | { file: $f.file_name, line: .line, type: .type }
      | "{\"file\":\"\(.file)\",\"line\":\(.line),\"smell\":\"S12\",\"severity\":\"H\",\"message\":\"mutation \(.type) survived; existing tests do not catch this behavior change\"}"
    ' "$GREMLINS_OUT" > "$S12_RAW" || true
else
    S12_RAW=""
fi

# --- Step 5: score ---
echo "Scoring findings..."
ALL_RAW="$(mktemp -t test-refine-all.XXXXXX)"
cat "$SMELLS_RAW" "$DUPES_RAW" "$DOMAIN_RAW" > "$ALL_RAW"
[[ -n "$S12_RAW" && -s "$S12_RAW" ]] && cat "$S12_RAW" >> "$ALL_RAW"

if [[ "$DO_COVERAGE" -eq 1 && -s "$COV_OUT" ]]; then
    "$SCORE_BIN" --weights "$WEIGHTS" --coverage "$COV_OUT" < "$ALL_RAW" > "$FINDINGS"
else
    "$SCORE_BIN" --weights "$WEIGHTS" < "$ALL_RAW" > "$FINDINGS"
fi

# --- Step 6: render report ---
echo "Rendering report..."
"$SCRIPTS/render-report.sh" \
    --findings "$FINDINGS" \
    --coverage "$COV_OUT" \
    --gremlins "$GREMLINS_OUT" \
    --scope "$SCOPE" \
    --slug "$SLUG" \
    --output "$REPORT"

# Cleanup tmp files (keep the FINDINGS_DIR ones).
rm -f "$SMELLS_RAW" "$DUPES_RAW" "$DOMAIN_RAW" "$ALL_RAW"
[[ -n "${S12_RAW:-}" ]] && rm -f "$S12_RAW"

echo
echo "Triage complete."
echo "  Report:   $REPORT"
echo "  Findings: $FINDINGS"
echo
echo "Review the report, check approved fixes, then run:"
echo "  $SCRIPTS/apply-fixes.sh --report $REPORT"
