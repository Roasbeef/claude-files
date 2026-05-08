#!/usr/bin/env bash
# Phase B: parse a refinement report, identify checked findings, and
# delegate the actual code edits to the calling agent (this script does
# not directly modify Go source — Edits go through the agent's Edit tool).
#
# This script:
#   1. Reads the report and extracts checked items (- [x]).
#   2. Joins them with the underlying findings JSON.
#   3. Emits a JSON action plan on stdout for the agent to consume.
#   4. After agent edits, the agent should re-invoke this with --post
#      to run go test and (optionally) gremlins for verification.
set -euo pipefail

usage() {
    cat <<EOF
Usage: $0 --report <md> [--post] [--verify-mutations] [--scope-pkg <path>]

Phases:
  default : Parse report, emit JSON action plan to stdout.
  --post  : Run 'go test ./... -race -count=1' and (with --verify-mutations)
            re-run gremlins to compute test_efficacy delta.
EOF
}

REPORT=""
POST=0
VERIFY_MUT=0
SCOPE_PKG="./..."
while [[ $# -gt 0 ]]; do
    case "$1" in
        --report) REPORT="$2"; shift 2 ;;
        --post) POST=1; shift ;;
        --verify-mutations) VERIFY_MUT=1; shift ;;
        --scope-pkg) SCOPE_PKG="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$REPORT" ]]; then
    usage >&2; exit 2
fi
if [[ ! -f "$REPORT" ]]; then
    echo "error: report not found: $REPORT" >&2
    exit 1
fi

REPORT_DIR="$(dirname "$REPORT")"
SLUG="$(basename "$REPORT" .md | sed 's/^[0-9]*-[0-9]*-[0-9]*-//')"
FINDINGS="$REPORT_DIR/_findings/$SLUG.json"

# --- Post phase: just run verification ---
if [[ "$POST" -eq 1 ]]; then
    echo "Running 'go test $SCOPE_PKG -race -count=1'..."
    if go test "$SCOPE_PKG" -race -count=1; then
        echo "go test: PASS"
    else
        echo "go test: FAIL — review changes before continuing" >&2
        exit 1
    fi

    if [[ "$VERIFY_MUT" -eq 1 ]]; then
        if ! command -v gremlins >/dev/null 2>&1; then
            echo "warn: gremlins not installed; skipping mutation verification" >&2
            exit 0
        fi
        before="$REPORT_DIR/_findings/$SLUG-gremlins.json"
        after="$REPORT_DIR/_findings/$SLUG-gremlins-after.json"
        echo "Re-running gremlins for verification..."
        "$HOME/.claude/skills/mutation-testing/scripts/unleash.sh" \
            --pkg "$SCOPE_PKG" --output "$after" --silent || true

        if [[ -s "$before" && -s "$after" ]] && command -v jq >/dev/null 2>&1; then
            ebefore="$(jq -r '.test_efficacy // 0' "$before")"
            eafter="$(jq -r '.test_efficacy // 0' "$after")"
            tbefore="$(jq -r '.mutants_total // 0' "$before")"
            tafter="$(jq -r '.mutants_total // 0' "$after")"
            # When either side produced zero mutants, the comparison is
            # not meaningful — report that explicitly instead of a
            # misleading "Δ -X%" derived from 0/0.
            if [[ "$tbefore" -eq 0 || "$tafter" -eq 0 ]]; then
                echo "test_efficacy: $ebefore% -> $eafter% (mutants: $tbefore -> $tafter; comparison invalid: zero-mutant side)"
                {
                    echo
                    echo "## After Refinement ($(date +'%Y-%m-%d %H:%M'))"
                    echo
                    echo "> ⚠ Mutation comparison invalid: one of the runs produced zero"
                    echo "> mutants (before=$tbefore, after=$tafter). Re-run gremlins"
                    echo "> manually to investigate before trusting these numbers."
                    echo
                    echo "| Metric | Before | After |"
                    echo "|---|---|---|"
                    echo "| test_efficacy | ${ebefore}% | ${eafter}% |"
                    echo "| mutants_total | ${tbefore} | ${tafter} |"
                } >> "$REPORT"
            else
                delta="$(awk -v a="$eafter" -v b="$ebefore" 'BEGIN{printf "%.2f", a - b}')"
                echo "test_efficacy: $ebefore% -> $eafter% (Δ $delta%)"
                {
                    echo
                    echo "## After Refinement ($(date +'%Y-%m-%d %H:%M'))"
                    echo
                    echo "| Metric | Before | After | Δ |"
                    echo "|---|---|---|---|"
                    echo "| test_efficacy | ${ebefore}% | ${eafter}% | ${delta}% |"
                    echo "| mutants_total | ${tbefore} | ${tafter} | |"
                } >> "$REPORT"
            fi
        fi
    fi
    exit 0
fi

# --- Default: parse report, emit action plan ---
if [[ ! -f "$FINDINGS" ]]; then
    echo "error: findings JSON not found: $FINDINGS" >&2
    exit 1
fi

# Extract approved item IDs from report (- [x] **Apply fix** lines under F<idx>).
# A finding F<N> is approved when the line `- [ ] **Apply fix**` is changed to
# `- [x] **Apply fix**` immediately following a `### F<N> — ...` heading.
APPROVED_F=()
APPROVED_REMOVE=()
APPROVED_RESHAPE=()
APPROVED_DOMAIN=()

current_section=""
current_finding=""
while IFS= read -r line; do
    # Section detection.
    case "$line" in
        "## Findings (Apply Approved)") current_section="findings" ;;
        "## Removal Candidates") current_section="removal" ;;
        "## Property-Based Testing Candidates") current_section="reshape" ;;
        "## Domain Checks") current_section="domain" ;;
        "## "*) current_section="other" ;;
    esac

    # Finding heading.
    if [[ "$current_section" == "findings" && "$line" =~ ^###\ F([0-9]+) ]]; then
        current_finding="${BASH_REMATCH[1]}"
        continue
    fi

    # Approval checkbox.
    if [[ "$line" == "- [x] **Apply fix**" && "$current_section" == "findings" && -n "$current_finding" ]]; then
        APPROVED_F+=("$current_finding")
        current_finding=""
    elif [[ "$line" =~ ^-\ \[x\]\ \*\*Remove\*\* && "$current_section" == "removal" ]]; then
        # Extract file:line and test name.
        APPROVED_REMOVE+=("$line")
    elif [[ "$line" =~ ^-\ \[x\]\ \*\*Reshape\*\* && "$current_section" == "reshape" ]]; then
        APPROVED_RESHAPE+=("$line")
    elif [[ "$line" =~ ^-\ \[x\]\  && "$current_section" == "domain" ]]; then
        APPROVED_DOMAIN+=("$line")
    fi
done < "$REPORT"

# Build action plan JSON. Bash with `set -u` errors on `${arr[@]}` when
# the array was never appended to; guard with `:-` to default to empty.
n_f=${#APPROVED_F[@]:-0}
n_remove=${#APPROVED_REMOVE[@]:-0}
n_reshape=${#APPROVED_RESHAPE[@]:-0}
n_domain=${#APPROVED_DOMAIN[@]:-0}
total=$(( n_f + n_remove + n_reshape + n_domain ))
if [[ "$total" -eq 0 ]]; then
    cat <<EOF
{
  "report": "$REPORT",
  "approved_count": 0,
  "message": "No fixes approved. Check boxes in the report and re-run."
}
EOF
    exit 0
fi

# Index findings by F<N> (1-based, matching report).
JQ_PROG='
  to_entries
  | map(.value + {f_id: (.key + 1 | tostring)})
'
ENRICHED="$(jq "$JQ_PROG" "$FINDINGS")"

# Resolve approved F<N> entries to full finding objects, then split
# by fix_kind so the consumer (the agent) knows which findings can be
# applied mechanically vs which need manual TODO comments.
APPROVED_FULL="["
first=1
for f in "${APPROVED_F[@]:-}"; do
    [[ -z "$f" ]] && continue
    match="$(echo "$ENRICHED" | jq --arg id "$f" '.[] | select(.f_id == $id)')"
    [[ -z "$match" || "$match" == "null" ]] && continue
    if [[ "$first" -eq 0 ]]; then APPROVED_FULL+=","; fi
    APPROVED_FULL+="$match"
    first=0
done
APPROVED_FULL+="]"

AUTO="$(echo "$APPROVED_FULL" | jq '[.[] | select((.fix_kind // "manual") == "auto")]')"
MANUAL="$(echo "$APPROVED_FULL" | jq '[.[] | select((.fix_kind // "manual") != "auto")]')"
AUTO_N="$(echo "$AUTO" | jq 'length')"
MANUAL_N="$(echo "$MANUAL" | jq 'length')"

# Print action plan.
{
    echo "{"
    echo "  \"report\": \"$REPORT\","
    echo "  \"findings_json\": \"$FINDINGS\","
    echo "  \"approved_count\": $total,"
    echo "  \"counts\": { \"auto\": $AUTO_N, \"manual\": $MANUAL_N },"
    echo "  \"guidance\": \"For 'auto' findings, apply the edit directly (delete the test or assertion). For 'manual' findings, write a TODO comment in the test file referencing this report path; the human/agent then writes the real test.\","
    echo "  \"approved_auto\": $AUTO,"
    echo "  \"approved_manual\": $MANUAL,"
    echo "  \"approved_removals\": ["
    first=1
    for r in "${APPROVED_REMOVE[@]:-}"; do
        [[ -z "$r" ]] && continue
        if [[ "$first" -eq 0 ]]; then echo ","; fi
        echo -n "    $(printf '%s' "$r" | jq -R .)"
        first=0
    done
    echo
    echo "  ],"
    echo "  \"approved_reshapes\": ["
    first=1
    for r in "${APPROVED_RESHAPE[@]:-}"; do
        [[ -z "$r" ]] && continue
        if [[ "$first" -eq 0 ]]; then echo ","; fi
        echo -n "    $(printf '%s' "$r" | jq -R .)"
        first=0
    done
    echo
    echo "  ],"
    echo "  \"approved_domain\": ["
    first=1
    for r in "${APPROVED_DOMAIN[@]:-}"; do
        [[ -z "$r" ]] && continue
        if [[ "$first" -eq 0 ]]; then echo ","; fi
        echo -n "    $(printf '%s' "$r" | jq -R .)"
        first=0
    done
    echo
    echo "  ]"
    echo "}"
} | jq .
