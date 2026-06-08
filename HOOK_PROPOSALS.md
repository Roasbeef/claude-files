# Hook Optimization Proposals

## Part 1: Updates to Existing Hooks

### 1. Context Enhancer (UserPromptSubmit) - SIMPLIFY

**Current Issues:**
- Too noisy with task management reminders
- Generic security/protocol context isn't always relevant
- Can clutter prompts unnecessarily

**Proposed Changes:**

```python
#!/usr/bin/env python3
"""
Simplified context enhancer - only adds critical context.
Focuses on security-critical and protocol-specific reminders.
"""

import json
import sys

def main():
    input_data = json.load(sys.stdin)
    prompt = input_data.get("prompt", "")

    context_additions = []

    # 1. Handle ultrathink mode (preserve existing)
    ultrathink_mode = False
    if prompt.rstrip().endswith("-u"):
        prompt = prompt.rstrip()[:-2].rstrip()
        ultrathink_mode = True

    # 2. ONLY add context for critical security keywords
    critical_security = [
        "race condition", "concurrent", "goroutine", "mutex", "atomic",
        "consensus", "validation", "mempool", "reorg", "double spend"
    ]
    if any(keyword in prompt.lower() for keyword in critical_security):
        context_additions.append(
            "⚠️ Security Critical: Consider race conditions, resource exhaustion, "
            "and attack vectors. Run with -race flag."
        )

    # 3. Bitcoin/Lightning protocol compliance reminders
    protocol_keywords = [
        "BIP", "BOLT", "TRUC", "v3 transaction", "package relay",
        "RBF", "CPFP", "taproot"
    ]
    if any(keyword in prompt.lower() for keyword in protocol_keywords):
        context_additions.append(
            "📋 Protocol: Verify BIP/BOLT compliance and test edge cases."
        )

    # Build enhanced prompt
    enhanced_prompt = prompt
    if context_additions:
        enhanced_prompt += "\n\n" + "\n".join(context_additions)

    if ultrathink_mode:
        enhanced_prompt += "\n\nultrathink"

    print(enhanced_prompt)
    sys.exit(0)

if __name__ == "__main__":
    main()
```

**Key Changes:**
- ✂️ Removed task management hints (noisy)
- ✂️ Removed generic testing reminders
- ✅ Kept only critical security reminders
- ✅ Kept protocol compliance reminders
- ✅ Cleaner, less noisy output

---

## Part 2: High-Value New Hooks for Agentic Coding

### Hook 1: Test Failure Analyzer (PostToolUse - Bash)

**Purpose:** Parse Go test failures into structured, actionable format so Claude can immediately fix issues.

**Triggers:** After any `go test` command

**Implementation:**

```bash
#!/bin/bash
# hooks/posttooluse/test_failure_analyzer.sh

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
command=$(echo "$input" | jq -r '.parameters.command // empty')

# Only run for Bash tool executing go test
if [[ "$tool" != "Bash" ]] || [[ ! "$command" =~ go[[:space:]]test ]]; then
    exit 0
fi

# Create temp file for analysis
output_file="/tmp/claude_test_output_$$"

# Capture test output (this runs after the tool, so we need to parse from logs)
# In practice, we'd need to capture the output from the Bash tool result
# For now, we'll document the pattern

echo ""
echo "🔍 Test Failure Analysis:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Parse test failures from output
# Look for patterns like:
# --- FAIL: TestFoo (0.01s)
#     foo_test.go:123: assertion failed
#     foo_test.go:124: expected 5, got 3

# Extract:
# 1. Failed test name
# 2. File and line numbers
# 3. Failure messages
# 4. Group related failures

# Output structured format:
echo "Failed Tests:"
echo "  1. TestChannelForceClose - channel_test.go:456"
echo "     Issue: Expected balance 1000, got 500"
echo "     Likely cause: Sweep transaction not accounting for fees"
echo ""
echo "  2. TestRaceCondition - peer_test.go:789"
echo "     Issue: Data race detected on peer.state"
echo "     Likely cause: Missing mutex lock in UpdateState()"
echo ""
echo "Suggested Fixes:"
echo "  • Add fee calculation in channel_test.go:456"
echo "  • Add mutex lock in peer.go:UpdateState()"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

exit 0
```

**Value:**
- ✅ Claude gets structured failure info instead of raw output
- ✅ Automatically groups related failures
- ✅ Suggests likely causes
- ✅ Provides file:line references for immediate fixing
- ✅ No need to parse hundreds of lines of test output

**Example Output:**
```
🔍 Test Failure Analysis:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Failed Tests:
  1. TestSweepTrigger - sweep_test.go:234
     Issue: timeout waiting for mempool entry
     Likely cause: Race condition, need WaitForMempoolEntry()

  2. TestReorgHandling - reorg_test.go:567
     Issue: channel balance incorrect after reorg
     Likely cause: Not re-validating transactions

Suggested Fixes:
  • Replace time.Sleep with WaitForMempoolEntry in sweep_test.go:234
  • Add transaction re-validation in reorg.go:handleReorg()
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

---

### Hook 2: Race Detector Parser (PostToolUse - Bash)

**Purpose:** Extract and format race condition details from `go test -race` output.

**Triggers:** After any `go test -race` command

**Implementation:**

```bash
#!/bin/bash
# hooks/posttooluse/race_detector_parser.sh

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
command=$(echo "$input" | jq -r '.parameters.command // empty')

# Only run for go test -race
if [[ "$tool" != "Bash" ]] || [[ ! "$command" =~ go[[:space:]]test.*-race ]]; then
    exit 0
fi

echo ""
echo "🏁 Race Condition Analysis:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Parse race detector output
# Pattern:
# ==================
# WARNING: DATA RACE
# Write at 0x00c000123456 by goroutine 7:
#   github.com/foo/bar.(*Peer).UpdateState()
#       /path/to/peer.go:123 +0x45
#
# Previous read at 0x00c000123456 by goroutine 12:
#   github.com/foo/bar.(*Peer).GetState()
#       /path/to/peer.go:89 +0x23

# Extract and format:
echo "Race Condition Detected:"
echo "  Location: peer.go:123 (UpdateState method)"
echo "  Type: Concurrent read/write on peer.state field"
echo ""
echo "Conflicting Access:"
echo "  Writer: goroutine 7 - Peer.UpdateState() at peer.go:123"
echo "  Reader: goroutine 12 - Peer.GetState() at peer.go:89"
echo ""
echo "Fix Strategy:"
echo "  1. Add sync.RWMutex to Peer struct"
echo "  2. Lock in UpdateState() before writing"
echo "  3. RLock in GetState() before reading"
echo ""
echo "Example Fix:"
echo "  type Peer struct {"
echo "      mu    sync.RWMutex"
echo "      state State"
echo "  }"
echo "  "
echo "  func (p *Peer) UpdateState(s State) {"
echo "      p.mu.Lock()"
echo "      defer p.mu.Unlock()"
echo "      p.state = s"
echo "  }"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

exit 0
```

**Value:**
- ✅ Parses complex race detector output into clear format
- ✅ Shows exactly which goroutines are conflicting
- ✅ Provides specific fix strategy with code example
- ✅ Claude can immediately implement the fix
- ✅ No need to understand raw race detector output

---

### Hook 3: Build Error Formatter (PostToolUse - Bash)

**Purpose:** Parse Go compiler errors into actionable groups.

**Triggers:** After `go build` or `go test` that fails to compile

**Implementation:**

```bash
#!/bin/bash
# hooks/posttooluse/build_error_formatter.sh

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
command=$(echo "$input" | jq -r '.parameters.command // empty')

# Only run for go build/test commands
if [[ "$tool" != "Bash" ]] || [[ ! "$command" =~ go[[:space:]](build|test) ]]; then
    exit 0
fi

# Check if command failed (would need access to exit code)
# For now, just document the pattern

echo ""
echo "🔨 Build Error Analysis:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Parse compiler errors and group by type:
# 1. Undefined references
# 2. Type mismatches
# 3. Missing imports
# 4. Syntax errors

echo "Error Summary (4 errors total):"
echo ""
echo "1. Undefined References (2 errors):"
echo "   • sweep.go:123 - undefined: Sweeper.TriggerSweep"
echo "   • sweep.go:145 - undefined: types.SweepRequest"
echo "   Fix: Add TriggerSweep method to Sweeper struct"
echo ""
echo "2. Type Mismatches (1 error):"
echo "   • channel.go:234 - cannot use int as btcutil.Amount"
echo "   Fix: Convert with btcutil.Amount(value)"
echo ""
echo "3. Missing Imports (1 error):"
echo "   • validator.go:45 - undefined: btcutil"
echo "   Fix: Add 'github.com/btcsuite/btcutil' import"
echo ""
echo "Priority Order:"
echo "  1. Fix missing imports first"
echo "  2. Add missing methods/types"
echo "  3. Fix type conversions"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

exit 0
```

**Value:**
- ✅ Groups related errors together
- ✅ Shows fix priority (imports → methods → conversions)
- ✅ Claude knows exactly what to fix and in what order
- ✅ Reduces back-and-forth fixing one error at a time

---

### Hook 4: Coverage Delta Tracker (PostToolUse - Bash)

**Purpose:** Track test coverage changes and show which lines gained/lost coverage.

**Triggers:** After `go test -cover` commands

**Implementation:**

```bash
#!/bin/bash
# hooks/posttooluse/coverage_delta_tracker.sh

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
command=$(echo "$input" | jq -r '.parameters.command // empty')

# Only run for go test with coverage
if [[ "$tool" != "Bash" ]] || [[ ! "$command" =~ go[[:space:]]test.*-cover ]]; then
    exit 0
fi

# Store previous coverage in temp file
prev_coverage_file="$HOME/.claude/coverage-cache/$(pwd | sed 's/\//_/g').txt"
mkdir -p "$HOME/.claude/coverage-cache"

# Get current coverage
current_coverage=$(go test -cover ./... 2>&1 | grep -o "coverage: [0-9.]*%" || echo "unknown")

echo ""
echo "📈 Coverage Analysis:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ -f "$prev_coverage_file" ]; then
    prev_coverage=$(cat "$prev_coverage_file")
    echo "Previous: $prev_coverage"
    echo "Current:  $current_coverage"
    echo ""

    # Calculate delta (would need proper parsing)
    echo "Changes:"
    echo "  • sweep.go: 75% → 85% (+10%)"
    echo "    New coverage: TriggerSweep method, error paths"
    echo "  • channel.go: 92% → 90% (-2%)"
    echo "    Lost coverage: handleReorg edge case (removed test?)"
    echo ""
    echo "Recommendation:"
    echo "  ✅ Good: sweep.go coverage improved"
    echo "  ⚠️  Action needed: Restore channel.go coverage"
else
    echo "First run: $current_coverage"
    echo "(No previous data for comparison)"
fi

# Save current coverage
echo "$current_coverage" > "$prev_coverage_file"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

exit 0
```

**Value:**
- ✅ Shows exactly which files gained/lost coverage
- ✅ Identifies what code is newly covered
- ✅ Warns if coverage regresses
- ✅ Claude knows if new tests are actually testing new code
- ✅ Tracks progress toward coverage goals

---

### Hook 5: Smart Lint Filter (PostToolUse - Edit/Write)

**Purpose:** Run linter but only show NEW issues introduced by recent changes.

**Triggers:** After Edit or Write on Go files

**Implementation:**

```bash
#!/bin/bash
# hooks/posttooluse/smart_lint_filter.sh

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
file_path=$(echo "$input" | jq -r '.parameters.file_path // empty')

# Only run for Go file edits
if [[ "$tool" != "Edit" && "$tool" != "Write" ]]; then
    exit 0
fi

if [[ ! "$file_path" =~ \.go$ ]]; then
    exit 0
fi

# Get current lint issues for this file
current_issues=$(golangci-lint run "$file_path" 2>&1 || true)

# Load previous issues from cache
cache_file="$HOME/.claude/lint-cache/$(echo "$file_path" | sed 's/\//_/g').txt"
mkdir -p "$HOME/.claude/lint-cache"

if [ -f "$cache_file" ]; then
    prev_issues=$(cat "$cache_file")

    # Find NEW issues (in current but not in previous)
    new_issues=$(comm -23 <(echo "$current_issues" | sort) <(echo "$prev_issues" | sort))

    if [ -n "$new_issues" ]; then
        echo ""
        echo "🔍 New Lint Issues in $file_path:"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "$new_issues"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "Fix these before committing."
    fi
fi

# Save current issues
echo "$current_issues" > "$cache_file"

exit 0
```

**Value:**
- ✅ Only shows NEW issues Claude introduced
- ✅ Doesn't clutter output with existing lint warnings
- ✅ Claude can fix issues immediately
- ✅ Prevents accumulating lint debt
- ✅ Focused, actionable feedback

---

### Hook 6: Benchmark Comparison (PostToolUse - Bash)

**Purpose:** Compare benchmark results before/after changes to show performance impact.

**Triggers:** After `go test -bench` commands

**Implementation:**

```bash
#!/bin/bash
# hooks/posttooluse/benchmark_comparison.sh

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
command=$(echo "$input" | jq -r '.parameters.command // empty')

# Only run for benchmark commands
if [[ "$tool" != "Bash" ]] || [[ ! "$command" =~ go[[:space:]]test.*-bench ]]; then
    exit 0
fi

# Store benchmark results
bench_cache="$HOME/.claude/benchmark-cache/$(pwd | sed 's/\//_/g').txt"
mkdir -p "$HOME/.claude/benchmark-cache"

echo ""
echo "⚡ Benchmark Analysis:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ -f "$bench_cache" ]; then
    echo "Comparison to Previous Run:"
    echo ""
    echo "BenchmarkValidateTransaction"
    echo "  Time:   1050ns/op → 980ns/op  (7% faster ✅)"
    echo "  Memory: 256B/op  → 256B/op   (no change)"
    echo "  Allocs: 4/op     → 3/op      (25% fewer ✅)"
    echo ""
    echo "BenchmarkMempool"
    echo "  Time:   5.2ms/op → 6.8ms/op  (31% slower ⚠️)"
    echo "  Memory: 1024B/op → 2048B/op  (100% more ❌)"
    echo "  Allocs: 12/op    → 24/op     (100% more ❌)"
    echo ""
    echo "Summary:"
    echo "  ✅ ValidateTransaction: Optimization successful"
    echo "  ❌ Mempool: Performance regression detected"
    echo "     Investigate: Likely the new transaction graph adds overhead"
    echo "     Action: Profile with pprof to find hotspots"
else
    echo "First benchmark run (no comparison data)"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

exit 0
```

**Value:**
- ✅ Shows performance impact of changes immediately
- ✅ Catches performance regressions before merge
- ✅ Validates optimizations actually work
- ✅ Shows memory and allocation changes
- ✅ Claude knows if changes are making things worse

---

### Hook 7: Concurrent Edit Detector (PreToolUse - Edit/Write)

**Purpose:** Detect if files have been modified outside Claude since they were read.

**Triggers:** Before Edit or Write operations

**Implementation:**

```bash
#!/bin/bash
# hooks/pretooluse/concurrent_edit_detector.sh

set -euo pipefail

input=$(cat)

tool=$(echo "$input" | jq -r '.tool // empty')
file_path=$(echo "$input" | jq -r '.parameters.file_path // empty')

# Only run for Edit/Write tools
if [[ "$tool" != "Edit" && "$tool" != "Write" ]]; then
    exit 0
fi

# Check if file exists
if [ ! -f "$file_path" ]; then
    exit 0
fi

# Track when files were last read
read_cache="$HOME/.claude/file-read-cache/$(echo "$file_path" | sed 's/\//_/g').txt"
mkdir -p "$HOME/.claude/file-read-cache"

if [ -f "$read_cache" ]; then
    last_read_time=$(cat "$read_cache")
    file_mod_time=$(stat -f "%m" "$file_path" 2>/dev/null || stat -c "%Y" "$file_path" 2>/dev/null)

    if [ "$file_mod_time" -gt "$last_read_time" ]; then
        echo ""
        echo "⚠️  WARNING: File Modified Externally"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "File: $file_path"
        echo ""
        echo "This file was modified outside Claude Code since it was last read."
        echo "The edit may conflict with external changes."
        echo ""
        echo "Recommendation: Re-read the file before editing to see latest changes."
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        # Don't block, just warn
    fi
fi

exit 0
```

**Value:**
- ✅ Prevents edit conflicts with external editors
- ✅ Warns Claude to re-read before editing
- ✅ Avoids overwriting user's external changes
- ✅ Especially useful when switching between IDE and Claude

---

## Part 3: Implementation Priority

### Immediate High Value (Implement First)

1. **Test Failure Analyzer** - Most impactful for agentic coding
2. **Build Error Formatter** - Essential for quick iteration
3. **Smart Lint Filter** - Prevents accumulating technical debt

### High Value (Implement Soon)

4. **Race Detector Parser** - Critical for concurrent code
5. **Concurrent Edit Detector** - Prevents conflicts

### Nice to Have (Implement Later)

6. **Coverage Delta Tracker** - Good for test-driven development
7. **Benchmark Comparison** - Good for performance work

---

## Part 4: Technical Considerations

### Challenge: Accessing Tool Output

The main challenge is that PostToolUse hooks receive tool **metadata** but not the actual **output** of the tool. To make these hooks work effectively, we need:

**Option A: Parse from conversation context**
- Hooks could look at recent conversation history
- Complex and fragile

**Option B: Store tool output in temp files**
- Modify how Bash tool stores output
- Hooks read from temp files
- More reliable

**Option C: Use environment variables**
- Set `CLAUDE_LAST_OUTPUT` env var with recent output
- Hooks read from env
- Clean and simple

**Recommendation: Option C** - Use environment variables to pass tool output to hooks.

### Example Pattern

```bash
# In PostToolUse hook
if [ -n "$CLAUDE_TOOL_OUTPUT" ]; then
    # Parse the output
    if echo "$CLAUDE_TOOL_OUTPUT" | grep -q "FAIL:"; then
        # Analyze test failures
    fi
fi
```

---

## Summary

### What We're Keeping
- ✅ Notification hooks (Stop, SubagentStop)
- ✅ ultrathink_hook
- ✅ sensitive_file_guard
- ✅ context_enhancer (simplified version)

### What We're Removing
- ❌ go_format_check (Claude knows to format)
- ❌ task_workflow_guard (noisy)
- ❌ load_project_context (you can see git/tasks yourself)
- ❌ save_session_context (low value)
- ❌ git_status_refresh (redundant)
- ❌ PreCompact hook (just captures git status)

### What We're Adding (High-Value)
1. Test Failure Analyzer - Parse test output into actions
2. Build Error Formatter - Group and prioritize compiler errors
3. Smart Lint Filter - Only show NEW lint issues
4. Race Detector Parser - Format race conditions with fixes
5. Concurrent Edit Detector - Prevent edit conflicts
6. Coverage Delta Tracker - Track coverage changes
7. Benchmark Comparison - Catch performance regressions

These hooks transform Claude from a passive assistant into an **active, aware developer** that can:
- Immediately understand what's broken
- Know exactly how to fix it
- Track progress and regressions
- Avoid conflicts and mistakes
