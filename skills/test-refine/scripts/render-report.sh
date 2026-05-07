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

while [[ $# -gt 0 ]]; do
    case "$1" in
        --findings) FINDINGS="$2"; shift 2 ;;
        --coverage) COVERAGE="$2"; shift 2 ;;
        --gremlins) GREMLINS="$2"; shift 2 ;;
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
# folded in. Avoids the "5 identical rows for one test" pattern from
# the agent's PR-305 feedback.
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

# Coverage summary line.
COV_LINE="(coverage analysis skipped)"
if [[ -n "$COVERAGE" && -s "$COVERAGE" ]]; then
    total="$(grep '^total:' "$COVERAGE" | awk '{print $NF}')"
    [[ -n "$total" ]] && COV_LINE="**Statement coverage**: $total"
fi

# Gremlins summary.
MUT_LINE="(mutation testing not run)"
if [[ -n "$GREMLINS" && -s "$GREMLINS" ]]; then
    eff="$(jq -r '.test_efficacy // 0' "$GREMLINS")"
    cov="$(jq -r '.mutations_coverage // 0' "$GREMLINS")"
    killed="$(jq -r '.mutants_killed // 0' "$GREMLINS")"
    lived="$(jq -r '.mutants_lived // 0' "$GREMLINS")"
    MUT_LINE="**Mutation testing**: efficacy=${eff}%, coverage=${cov}%, killed=${killed}, lived=${lived}"
fi

{
cat <<HEADER
# Test Refinement Report

**Date**: $DATE
**Scope**: $SCOPE
**Slug**: $SLUG

## Summary

- **Total findings**: $TOTAL ($H_COUNT high-severity, $M_COUNT medium, $L_COUNT low)
- **Confidence**: $HI_CONF high-confidence, $LO_CONF need manual verification
- $COV_LINE
- $MUT_LINE

> Each fix below has a checkbox. Check the box to approve the fix; leave unchecked to skip.
> When done, run \`apply-fixes.sh --report <this-file>\`. Only checked items are applied.
> **Test removals require an explicit checkbox under the "Removal Candidates" section.**

## Top Findings

| # | Priority | Conf | File:Line | Smell | Sev | Test | Message |
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
      "\(.value.file):\(.value.line)\(if (.value._dup_count // 1) > 1 then " (×\(.value._dup_count))" else "" end)",
      .value.smell,
      .value.severity,
      (.value.test_name // ""),
      ((.value.message // "") | .[0:70])
    ]
  | "| \(.[0]) | \(.[1]) | \(.[2]) | \(.[3]) | \(.[4]) | \(.[5]) | \(.[6]) | \(.[7]) |"
' "$FINDINGS"

cat <<'DETAIL_HEADER'

## Findings (Apply Approved)

DETAIL_HEADER

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

    cat <<DETAIL

### F$idx — \`$file:$line\` — $smell ($sev) — priority $pri, confidence $conf

- [ ] **Apply fix**
- **Test**: \`$test_name\`
- **Function under test**: \`$fut\`
- **Smell**: $msg
${sugg:+- **Suggestion**: $sugg}
$( [[ "$dups" -gt 1 ]] && echo "- **Note**: $dups identical findings collapsed (lines: $all_lines)" )

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
