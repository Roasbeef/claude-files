# Detector fixtures

Each subdirectory pressure-tests one smell ID. Layout:

```
<smell>/
    fixture_test.go          # synthetic Go tests exercising the detector
    expected.txt             # newline-separated "<line>:<smell>:<test_name>"
                             # entries the detector MUST produce for fixture_test.go
```

Fixtures intentionally include both true positives (TP) AND look-alikes
that *should not* fire (false positives the detector must reject).
Annotate each test in `fixture_test.go` with a `// TP:` or `// FP:`
comment explaining why.

## Running

```sh
~/.claude/skills/test-refine/testdata/run-detector-tests.sh
```

The runner builds detect-smells, runs it on each fixture file, extracts
`<line>:<smell>:<test_name>` triples from the JSON output, and diffs
against `expected.txt`. Exit non-zero on any mismatch.

`expected.txt` is sorted: one line per expected finding, sorted by
line number. Lines starting with `#` are comments. Empty file means
"no findings expected".
