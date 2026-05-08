#!/usr/bin/env bash
# Detector regression harness. Walks each subdir of testdata/, runs
# detect-smells.go (and friends) against fixture_test.go, and diffs
# the actual findings against expected.txt.
#
# Exit 0: all fixtures pass. Exit 1: any mismatch.
set -uo pipefail

SKILL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS="$SKILL_DIR/scripts"
DATA_DIR="$SKILL_DIR/testdata"
BIN_DIR="${TEST_REFINE_BIN_DIR:-$HOME/.claude/cache/test-refine-bin}"

build_analyzer() {
    local src="$1" name; name="$(basename "$src" .go)"
    local out="$BIN_DIR/$name"
    if [[ -x "$out" && "$out" -nt "$src" ]]; then
        echo "$out"; return
    fi
    mkdir -p "$BIN_DIR"
    local td; td="$(mktemp -d -t test-refine-build.XXXXXX)"
    cp "$src" "$td/"
    (
        cd "$td"
        cat > go.mod <<EOF
module local

go 1.22
EOF
        go build -o "$out" "./$(basename "$src")"
    ) >&2
    rm -rf "$td"
    echo "$out"
}

# normalize_findings prints `line:smell:test_name` triples sorted by line.
# Reads JSON-lines on stdin.
normalize_findings() {
    jq -r 'select(.smell != null) | "\(.line):\(.smell):\(.test_name // "")"' \
        | sort -t: -k1,1n -k2,2 -k3,3
}

# load_expected prints sorted expected triples from expected.txt,
# stripping comments + blank lines.
load_expected() {
    local f="$1"
    [[ -f "$f" ]] || { echo ""; return; }
    grep -vE '^\s*(#|$)' "$f" | sort -t: -k1,1n -k2,2 -k3,3
}

SMELLS_BIN="$(build_analyzer "$SCRIPTS/detect-smells.go")"
DUPES_BIN="$(build_analyzer "$SCRIPTS/detect-duplicates.go")"

fail_count=0
pass_count=0

for fix_dir in "$DATA_DIR"/*/; do
    name="$(basename "$fix_dir")"
    [[ "$name" == "_archive" ]] && continue
    fixture="$fix_dir/fixture_test.go"
    expected="$fix_dir/expected.txt"
    [[ -f "$fixture" ]] || continue

    # Detector context: include any sibling production .go files in the
    # same dir (e.g. helpers/<package>.go) so the package-wide function
    # index is populated.
    inputs=("$fixture")
    while IFS= read -r f; do
        [[ -n "$f" && -f "$f" ]] && inputs+=("$f")
    done < <(find "$fix_dir" -maxdepth 1 -type f -name '*.go' ! -name 'fixture_test.go')

    # Combine smell + duplicate detector outputs. The s08 fixture
    # exercises detect-duplicates; other fixtures only exercise
    # detect-smells. Merging both keeps the harness simple.
    actual="$( {
        "$SMELLS_BIN" "${inputs[@]}"
        "$DUPES_BIN" "${inputs[@]}"
    } 2>/dev/null | normalize_findings )"
    want="$(load_expected "$expected")"

    if [[ "$actual" == "$want" ]]; then
        echo "PASS  $name"
        pass_count=$((pass_count + 1))
    else
        echo "FAIL  $name"
        diff <(printf '%s\n' "$want") <(printf '%s\n' "$actual") \
            | sed 's/^/        /'
        fail_count=$((fail_count + 1))
    fi
done

echo
echo "Result: $pass_count passed, $fail_count failed."
[[ "$fail_count" -eq 0 ]]
