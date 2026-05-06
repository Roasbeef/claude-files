#!/usr/bin/env bash
# Parse a gremlins JSON report and emit a human-readable survivor summary.
# Groups LIVED mutants by file and mutation type, highlighting consensus-
# critical paths first when present.
set -euo pipefail

usage() {
    cat <<EOF
Usage: $0 --input <gremlins.json> [--output <markdown>] [--top N]

  --input   Path to gremlins JSON report (from 'gremlins unleash --output').
  --output  Markdown file to write (default: stdout).
  --top     Show top N survivors by file (default: 50).

Output sections:
  - Summary (efficacy, mutations_coverage, totals)
  - Survivors by file (line, column, mutator type)
  - Risk-weighted ordering (consensus/channel/wallet/payment paths first)
EOF
}

INPUT=""
OUTPUT=""
TOP=50

while [[ $# -gt 0 ]]; do
    case "$1" in
        --input)  INPUT="$2"; shift 2 ;;
        --output) OUTPUT="$2"; shift 2 ;;
        --top)    TOP="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "unknown flag: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$INPUT" ]]; then
    echo "error: --input is required" >&2
    usage >&2
    exit 2
fi

if ! command -v jq >/dev/null 2>&1; then
    echo "error: jq is required for analyze-survivors.sh" >&2
    exit 1
fi

emit() {
    if [[ -n "$OUTPUT" ]]; then
        cat >> "$OUTPUT"
    else
        cat
    fi
}

if [[ -n "$OUTPUT" ]]; then
    : > "$OUTPUT"
fi

{
    module="$(jq -r '.go_module // "unknown"' "$INPUT")"
    efficacy="$(jq -r '.test_efficacy // 0' "$INPUT")"
    coverage="$(jq -r '.mutations_coverage // 0' "$INPUT")"
    total="$(jq -r '.mutants_total // 0' "$INPUT")"
    killed="$(jq -r '.mutants_killed // 0' "$INPUT")"
    lived="$(jq -r '.mutants_lived // 0' "$INPUT")"
    notcov="$(jq -r '.mutants_not_covered // 0' "$INPUT")"
    notvia="$(jq -r '.mutants_not_viable // 0' "$INPUT")"
    elapsed="$(jq -r '.elapsed_time // 0' "$INPUT")"

    cat <<EOF
# Gremlins Survivor Analysis

**Module**: $module
**Source**: $INPUT

## Summary

| Metric | Value |
|---|---|
| Test efficacy | ${efficacy}% |
| Mutations coverage | ${coverage}% |
| Mutants killed | $killed |
| Mutants LIVED | $lived |
| Mutants not covered | $notcov |
| Mutants not viable | $notvia |
| Total mutants | $total |
| Elapsed time | ${elapsed}s |

EOF

    if [[ "$lived" == "0" || "$lived" == "null" ]]; then
        echo "All covered mutants were killed. No survivors to analyze."
        exit 0
    fi

    echo "## Survivors by File"
    echo
    echo "| File | Line:Col | Mutator |"
    echo "|---|---|---|"

    # Risk-weighted: prioritize files matching critical paths.
    jq -r '
      .files[]
      | . as $f
      | .mutations[]
      | select(.status == "LIVED")
      | [$f.file_name, "\(.line):\(.column)", .type] | @tsv
    ' "$INPUT" \
    | awk -F'\t' '{
        risk = 5
        if ($1 ~ /(consensus|channel|commit|payment|crypto|sign|verify|wallet|htlc|scoring|invoice)/) risk = 1
        else if ($1 ~ /internal\//) risk = 2
        else if ($1 ~ /cmd\//) risk = 3
        printf "%d\t%s\t%s\t%s\n", risk, $1, $2, $3
      }' \
    | sort -k1,1n -k2,2 \
    | head -n "$TOP" \
    | awk -F'\t' '{ printf "| %s | %s | %s |\n", $2, $3, $4 }'

    echo
    echo "## Mutator Breakdown (LIVED)"
    echo
    echo "| Mutator | Count |"
    echo "|---|---|"
    jq -r '
      [.files[].mutations[] | select(.status == "LIVED") | .type]
      | group_by(.)
      | map({type: .[0], count: length})
      | sort_by(-.count)[]
      | "\(.type)\t\(.count)"
    ' "$INPUT" | awk -F'\t' '{ printf "| %s | %d |\n", $1, $2 }'

    echo
    echo "## Suggested Next Steps"
    echo
    echo "- For each surviving mutator, identify the function it targets and add an assertion that distinguishes the original from the mutated value."
    echo "- For mutators in critical paths (top of the table), aim for 100% kill rate before merging."
    echo "- See \`references/best_practices.md\` for category-specific test patterns."
} | emit
