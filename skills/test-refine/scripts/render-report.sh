#!/usr/bin/env bash
# Render a markdown refinement report from scored findings JSON.
set -euo pipefail

usage() {
    cat <<EOF
Usage: $0 --findings <json> --output <md> [other options]

  --findings PATH   Scored findings JSON (sorted array; from score.go).
  --coverage PATH   go tool cover -func output (optional).
  --gremlins PATH   Gremlins JSON output (optional).
  --scope NAME      Scope label (e.g. "package").
  --slug NAME       Filename slug.
  --output PATH     Markdown report output path.
  --top N           Show top N findings in detail (default 30).
EOF
}

FINDINGS=""
COVERAGE=""
GREMLINS=""
SCOPE=""
SLUG=""
OUTPUT=""
TOP=30
COV_REQUESTED=0
MUT_REQUESTED=0

while [[ $# -gt 0 ]]; do
    case "$1" in
        --findings) FINDINGS="$2"; shift 2 ;;
        --coverage) COVERAGE="$2"; shift 2 ;;
        --gremlins) GREMLINS="$2"; shift 2 ;;
        --coverage-requested) COV_REQUESTED=1; shift ;;
        --mutations-requested) MUT_REQUESTED=1; shift ;;
        --scope) SCOPE="$2"; shift 2 ;;
        --slug) SLUG="$2"; shift 2 ;;
        --output) OUTPUT="$2"; shift 2 ;;
        --top) TOP="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$FINDINGS" || -z "$OUTPUT" ]]; then
    usage >&2; exit 2
fi
if ! command -v jq >/dev/null 2>&1; then
    echo "error: jq is required" >&2
    exit 1
fi

DATE="$(date +'%Y-%m-%d %H:%M')"

# Collapse identical (file, test_name, smell) findings into one entry,
# keeping the highest-priority instance and tracking how many were
# folded in. Avoids "N identical rows for one test" patterns where a
# single helper produces several findings at distinct line numbers.
COLLAPSED="$(mktemp -t test-refine-collapsed.XXXXXX)"
jq '
  group_by([.file, .test_name, .smell])
  | map(
      (max_by(.priority // 0)) + {
        _dup_count: length,
        _all_lines: (map(.line) | unique | sort)
      }
    )
  | sort_by(-(.priority // 0))
' "$FINDINGS" > "$COLLAPSED"
FINDINGS_RAW="$FINDINGS"
FINDINGS="$COLLAPSED"

TOTAL="$(jq 'length' "$FINDINGS")"
H_COUNT="$(jq '[.[] | select(.severity == "H")] | length' "$FINDINGS")"
M_COUNT="$(jq '[.[] | select(.severity == "M")] | length' "$FINDINGS")"
L_COUNT="$(jq '[.[] | select(.severity == "L")] | length' "$FINDINGS")"

# Confidence split (B): findings with confidence >= 0.7 are
# "high-confidence". Lower-confidence findings need manual verification.
HI_CONF="$(jq '[.[] | select((.confidence // 1.0) >= 0.7)] | length' "$FINDINGS")"
LO_CONF="$(jq '[.[] | select((.confidence // 1.0) < 0.7)] | length' "$FINDINGS")"

# Fix-kind split (DEF3): "auto" findings can be mechanically applied
# (e.g., delete a tautology); "manual" findings require writing real
# test code or understanding intent.
AUTO_FIX="$(jq '[.[] | select((.fix_kind // "manual") == "auto")] | length' "$FINDINGS")"
MANUAL_FIX="$(jq '[.[] | select((.fix_kind // "manual") != "auto")] | length' "$FINDINGS")"

# Coverage summary line + data-quality flag.
COV_LINE="(coverage analysis skipped)"
COV_BROKEN=0
if [[ -n "$COVERAGE" && -s "$COVERAGE" ]]; then
    total="$(grep '^total:' "$COVERAGE" | awk '{print $NF}')"
    if [[ -n "$total" ]]; then
        COV_LINE="**Statement coverage**: $total"
        # 0.0% with coverage requested almost always means the build
        # failed and the script swallowed the error. Flag it.
        if [[ "$total" == "0.0%" && "$COV_REQUESTED" -eq 1 ]]; then
            COV_BROKEN=1
        fi
    fi
elif [[ "$COV_REQUESTED" -eq 1 ]]; then
    COV_LINE="**Statement coverage**: data missing (coverage step failed)"
    COV_BROKEN=1
fi

# Gremlins summary + data-quality flag.
MUT_LINE="(mutation testing not run)"
MUT_BROKEN=0
if [[ -n "$GREMLINS" && -s "$GREMLINS" ]]; then
    eff="$(jq -r '.test_efficacy // 0' "$GREMLINS")"
    cov="$(jq -r '.mutations_coverage // 0' "$GREMLINS")"
    killed="$(jq -r '.mutants_killed // 0' "$GREMLINS")"
    lived="$(jq -r '.mutants_lived // 0' "$GREMLINS")"
    MUT_LINE="**Mutation testing**: efficacy=${eff}%, coverage=${cov}%, killed=${killed}, lived=${lived}"
elif [[ "$MUT_REQUESTED" -eq 1 ]]; then
    MUT_LINE="**Mutation testing**: data missing (gremlins requested but produced no output)"
    MUT_BROKEN=1
fi

# Pre-flight sampler banner (DEF4): on a sample of the top findings,
# count how many are high-confidence. If under 60% of the sample is
# high-confidence, the report likely has a high false-positive rate
# and the user should manually verify before acting.
SAMPLE_SIZE=5
[[ "$TOTAL" -lt "$SAMPLE_SIZE" ]] && SAMPLE_SIZE="$TOTAL"
BANNER=""
if [[ "$SAMPLE_SIZE" -gt 0 ]]; then
    PLAUSIBLE="$(jq --argjson n "$SAMPLE_SIZE" \
        '[.[:$n][] | select((.confidence // 1.0) >= 0.7)] | length' "$FINDINGS")"
    # Threshold: < 60% (i.e., fewer than 3 of 5).
    threshold=$(( SAMPLE_SIZE * 60 / 100 ))
    if [[ "$PLAUSIBLE" -lt "$threshold" ]]; then
        BANNER+="
> ⚠ **Spot-check warning**: only $PLAUSIBLE of the top $SAMPLE_SIZE findings are
> high-confidence. The report likely has a high false-positive rate.
> Manually verify each finding before acting; consider re-running with
> a narrower scope or with --use-mutations for stronger signal.
"
    fi
fi

# Data-layer warnings: coverage and/or gremlins were requested but
# produced no usable signal. Priority scores collapse to severity-only
# under these conditions, so the user must know not to trust the ranks.
# Branch-gap degraded: every finding has gap==0. Same effective signal
# as the score.go renormalize warning, but surfaced where users see it.
GAP_DEGRADED=0
if [[ "$TOTAL" -gt 0 ]]; then
    NONZERO_GAP="$(jq '[.[] | select((.gap // 0) > 0)] | length' "$FINDINGS")"
    if [[ "$NONZERO_GAP" -eq 0 ]]; then
        GAP_DEGRADED=1
    fi
fi
if [[ "$GAP_DEGRADED" -eq 1 ]]; then
    BANNER+="
> ⚠ **Priority is severity-only**: branch-gap data was unavailable for
> every finding (coverage probably didn't link function names to
> production sources). Priority numbers are derived from path-risk and
> severity alone — different findings with the same severity will
> share priority. Treat the ranking as coarse.
"
fi

if [[ "$COV_BROKEN" -eq 1 || "$MUT_BROKEN" -eq 1 ]]; then
    BANNER+="
> ⚠ **Data layer was incomplete**:"
    [[ "$COV_BROKEN" -eq 1 ]] && BANNER+="
> - Coverage data is missing or 0.0% across the board. Most likely the
>   \`go test -cover\` step failed to build (uninitialised submodules,
>   missing build tags, etc.). The branch-gap component of priority is
>   noise on this run."
    [[ "$MUT_BROKEN" -eq 1 ]] && BANNER+="
> - Mutation data is missing despite \`--use-mutations\`. Gremlins
>   either failed to install, failed to download dependencies, or was
>   silently dropped (e.g. on diff/repo scope). S12 findings will not
>   appear; trust mutation-derived priority cautiously."
    BANNER+="
>
> Treat priority numbers as advisory rather than authoritative for this report.
"
fi

{
cat <<HEADER
# Test Refinement Report

**Date**: $DATE
**Scope**: $SCOPE
**Slug**: $SLUG
$BANNER
## Summary

- **Total findings**: $TOTAL ($H_COUNT high-severity, $M_COUNT medium, $L_COUNT low)
- **Confidence**: $HI_CONF high-confidence, $LO_CONF need manual verification
- **Fix kind**: $AUTO_FIX auto (mechanical), $MANUAL_FIX manual (needs new test or intent)
- $COV_LINE
- $MUT_LINE

> Each fix below has a checkbox. Check the box to approve the fix; leave unchecked to skip.
> When done, run \`apply-fixes.sh --report <this-file>\`. Only checked items are applied.
> **Test removals require an explicit checkbox under the "Removal Candidates" section.**

## Top Findings

Full message + suggestion + test body are in the detail block for each
finding below (\`F\${n}\`). The table is for at-a-glance scanning only.

| # | Priority | Conf | Fix | File:Line | Smell | Sev | Test / SUT |
|---|---|---|---|---|---|---|---|
HEADER

jq -r --argjson top "$TOP" '
  to_entries
  | map(. + {idx: (.key + 1)})
  | .[:$top]
  | .[]
  | [
      .idx,
      (.value.priority // 0 | . * 100 | floor / 100),
      ((.value.confidence // 1.0) | . * 100 | floor / 100),
      (.value.fix_kind // "manual"),
      "\(.value.file):\(.value.line)\(if (.value._dup_count // 1) > 1 then " (×\(.value._dup_count))" else "" end)",
      .value.smell,
      .value.severity,
      (
        if (.value.test_name // "") != "" and (.value.function_under_test // "") != ""
        then "\(.value.test_name) → \(.value.function_under_test)"
        elif (.value.test_name // "") != ""
        then .value.test_name
        else (.value.function_under_test // "")
        end
      )
    ]
  | "| \(.[0]) | \(.[1]) | \(.[2]) | \(.[3]) | \(.[4]) | \(.[5]) | \(.[6]) | \(.[7]) |"
' "$FINDINGS"

cat <<'DETAIL_HEADER'

## Findings (Apply Approved)

DETAIL_HEADER

# extract_test_body prints the test function (or surrounding 30 lines)
# from the source file as a numbered code block. Heuristic: starts at
# the finding's line and stops at the first `^}` (column-0 closing
# brace) or 30 lines, whichever comes first.
extract_test_body() {
    local file="$1" start="$2" max=30
    [[ -z "$file" || -z "$start" ]] && return 0
    [[ -f "$file" ]] || return 0
    awk -v start="$start" -v max="$max" '
      NR >= start {
        printed++
        # Print line number (right-padded width 5) + content.
        printf "%5d  %s\n", NR, $0
        # Stop at column-0 closing brace (top-level fn end) or budget.
        if ($0 ~ /^}/ && printed > 1) exit
        if (printed >= max) { print "       ... (truncated)"; exit }
      }
    ' "$file"
}

# Detail blocks for each finding.
jq -c --argjson top "$TOP" '.[:$top] | .[]' "$FINDINGS" | nl -ba | while IFS=$'\t' read -r num row; do
    idx="$(echo "$num" | tr -d ' ')"
    file="$(echo "$row"  | jq -r '.file')"
    line="$(echo "$row"  | jq -r '.line')"
    smell="$(echo "$row" | jq -r '.smell')"
    sev="$(echo "$row"   | jq -r '.severity')"
    msg="$(echo "$row"   | jq -r '.message')"
    test_name="$(echo "$row" | jq -r '.test_name // ""')"
    fut="$(echo "$row"   | jq -r '.function_under_test // ""')"
    sugg="$(echo "$row"  | jq -r '.suggestion // ""')"
    conf="$(echo "$row"  | jq -r '(.confidence // 1.0) | . * 100 | floor / 100')"
    dups="$(echo "$row"  | jq -r '._dup_count // 1')"
    all_lines="$(echo "$row" | jq -r '(._all_lines // [.line]) | join(", ")')"
    pri="$(echo "$row"   | jq -r '(.priority // 0) | . * 100 | floor / 100')"

    # Removal smells go in the dedicated section below; here we emit
    # strengthening/reshape/add findings.
    case "$smell" in
        S03|S08)
            # Removal candidates handled separately.
            continue
            ;;
    esac

    body_block=""
    if [[ -f "$file" ]]; then
        body_text="$(extract_test_body "$file" "$line")"
        if [[ -n "$body_text" ]]; then
            body_block=$'\n<details><summary>show test body</summary>\n\n```go\n'"$body_text"$'\n```\n\n</details>\n'
        fi
    fi

    cat <<DETAIL

### F$idx — \`$file:$line\` — $smell ($sev) — priority $pri, confidence $conf

- [ ] **Apply fix**
- [ ] **False positive — won't fix** (record disagreement; survives across re-runs)
- **Test**: \`$test_name\`
- **Function under test**: \`$fut\`
- **Smell**: $msg
${sugg:+- **Suggestion**: $sugg}
$( [[ "$dups" -gt 1 ]] && echo "- **Note**: $dups identical findings collapsed (lines: $all_lines)" )
$body_block
DETAIL
done

cat <<'REMOVAL_HEADER'

## Removal Candidates

These tests were flagged as trivial or duplicate. Removal requires an explicit checkbox.

REMOVAL_HEADER

REMOVABLE="$(jq '[.[] | select(.smell == "S03" or .smell == "S08" or .smell == "S01")]' "$FINDINGS")"
COUNT="$(echo "$REMOVABLE" | jq 'length')"
if [[ "$COUNT" -eq 0 ]]; then
    echo "_No removal candidates._"
else
    echo "$REMOVABLE" | jq -r '
      .[]
      | "- [ ] **Remove** `\(.file):\(.line)` — `\(.test_name)` — \(.smell): \(.message)"
    '
fi

cat <<'PBT_HEADER'

## Property-Based Testing Candidates

Functions with patterns suggesting a property-based rewrite (rapid).

PBT_HEADER

PBT="$(jq '[.[] | select(.smell == "D-PBT-CANDIDATE")]' "$FINDINGS")"
PBT_COUNT="$(echo "$PBT" | jq 'length')"
if [[ "$PBT_COUNT" -eq 0 ]]; then
    echo "_No PBT candidates detected._"
else
    echo "$PBT" | jq -r '
      .[]
      | "- [ ] **Reshape** `\(.function_under_test)` — \(.message)\n      Suggested property: \(.suggestion)"
    '
fi

cat <<'DOMAIN_HEADER'

## Domain Checks

DOMAIN_HEADER

for code in D-CONCURRENCY-MISSING D-ERR-PATH-MISSING D-CTX-CANCEL-MISSING D-CTX-TIMEOUT-MISSING D-DETERMINISM-CLOCK D-DETERMINISM-RAND D-DETERMINISM-ENV; do
    matches="$(jq --arg c "$code" '[.[] | select(.smell == $c)]' "$FINDINGS")"
    n="$(echo "$matches" | jq 'length')"
    if [[ "$n" -gt 0 ]]; then
        echo "### $code ($n)"
        echo
        echo "$matches" | jq -r '
          .[]
          | "- [ ] `\(.file):\(.line)` — `\(.function_under_test // .test_name)` — \(.message)"
        '
        echo
    fi
done

cat <<APPLY_FOOTER

---

## Next Step

\`\`\`bash
~/.claude/skills/test-refine/scripts/apply-fixes.sh \\
    --report $OUTPUT
\`\`\`

Add \`--verify-mutations\` to re-run gremlins after fixes and confirm
\`test_efficacy\` did not regress.
APPLY_FOOTER
} > "$OUTPUT"

# Clean up the temp collapsed JSON; keep the original findings file.
rm -f "$COLLAPSED"

echo "Report written: $OUTPUT"
