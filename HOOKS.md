# Claude Code Hooks Configuration

This document describes all the hooks configured in your Claude Code environment and how to use/customize them.

## Overview

Hooks are shell commands that execute at various points in Claude Code's lifecycle. They provide deterministic control over behavior, enable automation, and help maintain consistency across projects.

**Hook Guide**: https://docs.claude.com/en/docs/claude-code/hooks-guide

## Currently Active Hooks

### 🎯 UserPromptSubmit Hooks

Runs when you submit a prompt to Claude.

#### 1. Ultrathink Hook (Original)
**File**: `hooks/ultrathink_hook.py`

**What it does**: Enables "ultrathink" mode when you end your prompt with `-u`

**Usage**: Add `-u` to the end of any prompt
```
Fix this bug in the transaction validator -u
```

#### 2. Context Enhancer Hook (NEW)
**File**: `hooks/userpromptsubmit/context_enhancer.py`

**What it does**: Automatically adds relevant context to your prompts based on keywords
- Detects task-related prompts → Suggests using /task-next if no active task
- Detects security keywords → Adds security considerations
- Detects Bitcoin/Lightning keywords → Adds protocol context
- Detects testing keywords → Adds testing guidance

**Examples**:
```
Prompt: "implement transaction validation"
→ Adds: "Consider using /task-next..." + "Bitcoin/Lightning Context..." + "Testing Context..."

Prompt: "fix the race condition in peer connection"
→ Adds: "Security Context: DoS vectors, race conditions..."
```

**Customization**: Edit the keyword lists in the Python script to match your workflow.

---

### 🛡️ PreToolUse Hooks

Runs before Claude executes a tool. Can provide feedback or block execution.

#### 1. Sensitive File Guard
**File**: `hooks/pretooluse/sensitive_file_guard.sh`

**What it does**: Protects sensitive files from accidental modification
- **BLOCKS**: .env files, credentials, keys, certificates
- **WARNS**: config files

**Blocked patterns**:
- `.env`, `.env.*`
- `credentials.json`
- `*.pem`, `*.key`, `*.p12`, `*.pfx`
- `id_rsa`, `id_rsa.pub`

**To temporarily disable**:
```bash
# Edit settings.json and comment out the hook
```

#### 2. Task Workflow Guard
**File**: `hooks/pretooluse/task_workflow_guard.sh`

**What it does**: Reminds Claude when editing files without an active task in your task management system

**When it triggers**: When Claude tries to Edit/Write files and no task is marked as `in_progress`

**Output**: "💡 Reminder: No task currently in_progress. Consider using /task-next"

#### 3. Go Format Check
**File**: `hooks/pretooluse/go_format_check.sh`

**What it does**: Checks if Go files need formatting before edits

**Output**: "⚠️ Note: file.go needs Go formatting (gofmt)"

**Note**: This is non-blocking - it only provides feedback.

---

### ✅ PostToolUse Hooks

Runs after Claude completes a tool execution.

#### 1. Git Status Refresh (ENABLED)
**File**: `hooks/posttooluse/git_status_refresh.sh`

**What it does**: Shows git status after file modifications
- Shows count of modified files
- Shows `git status --short` for ≤5 files
- Suggests running `git status` for >5 files

**Example output**:
```
📊 Git Status: 3 file(s) modified
 M src/validator.go
 M src/validator_test.go
?? docs/design.md
```

#### 2. Go Test Runner (DISABLED by default)
**File**: `hooks/posttooluse/go_test_runner.sh`

**What it does**: Automatically runs tests after Go code changes

**To enable**:
```bash
# Set environment variable before starting Claude Code
export CLAUDE_AUTO_TEST=1
claude-code
```

**Behavior**:
- Runs tests in the same package as the modified file
- Uses 30s timeout
- Shows pass/fail status

**Warning**: May slow down workflow. Enable only when focusing on TDD.

#### 3. Coverage Tracker (DISABLED by default)
**File**: `hooks/posttooluse/coverage_tracker.sh`

**What it does**: Tracks test coverage after code changes

**To enable**:
```bash
export CLAUDE_TRACK_COVERAGE=1
claude-code
```

**Output**: Shows coverage percentage for the modified package

---

### 🚀 SessionStart Hook

Runs when you start a Claude Code session.

**File**: `hooks/sessionstart/load_project_context.sh`

**What it does**: Loads project context and shows key information
- Current directory and git branch
- Uncommitted changes count
- Task system status (if `.tasks/` exists)
- Project type detection (Go/Node/Rust)
- Last session summary (if available)

**Example output**:
```
🚀 Claude Code Session Starting

📁 Directory: lnd
🌿 Git Branch: master
   Uncommitted changes: 3 file(s)

📋 Task Management System Active
   Total Tasks: 5
   ⚙️  In Progress: 1
   ✅ Ready: 4

   💡 Use /task-view to see current task details

🔵 Go Project Detected
   Go Version: go1.21.5
```

---

### 💾 SessionEnd Hook

Runs when you end a Claude Code session.

**File**: `hooks/sessionend/save_session_context.sh`

**What it does**: Saves session context for next time
- Saves modified files list
- Notes in-progress tasks
- Creates session summary

**Saved to**: `~/.claude/session-context/{project}_last_session.txt`

**Benefit**: Next session starts with context from last session.

---

### 📦 PreCompact Hook

Runs before Claude compacts the conversation history.

**File**: `hooks/precompact/save_important_context.sh`

**What it does**: Archives important context before compaction
- Current git state
- In-progress and blocked tasks
- Recent documentation
- Key decisions

**Saved to**: `~/.claude/compaction-archives/{project}_compact_{timestamp}.txt`

**Benefit**: No important context is lost during compaction.

---

### 🔔 Notification Hooks (Original)

These were already configured and remain unchanged.

#### Stop Hook
**What it does**: Plays sound and shows alert when Claude finishes responding

#### SubagentStop Hook
**What it does**: Notifies when subagent tasks complete

#### Notification Hook
**What it does**: General notifications from Claude Code

---

## Hook Management

### Enabling/Disabling Hooks

Edit `~/.claude/settings.json`:

**To disable a hook**: Remove or comment out its entry
```json
"PreToolUse": [
  {
    "matcher": "",
    "hooks": [
      // {
      //   "type": "command",
      //   "command": "/Users/roasbeef/.claude/hooks/pretooluse/go_format_check.sh"
      // }
    ]
  }
]
```

**To enable optional hooks**: Set environment variables
```bash
export CLAUDE_AUTO_TEST=1          # Enable auto-test after code changes
export CLAUDE_TRACK_COVERAGE=1     # Enable coverage tracking
```

### Creating Custom Hooks

1. **Create a new script** in the appropriate hooks directory:
   ```bash
   touch ~/.claude/hooks/pretooluse/my_custom_hook.sh
   chmod +x ~/.claude/hooks/pretooluse/my_custom_hook.sh
   ```

2. **Write the hook script**:
   ```bash
   #!/bin/bash
   # Read input from stdin
   input=$(cat)

   # Extract data with jq
   tool=$(echo "$input" | jq -r '.tool // empty')

   # Your logic here

   # Exit 0 to continue, exit 1 to block
   exit 0
   ```

3. **Add to settings.json**:
   ```json
   "PreToolUse": [
     {
       "matcher": "",
       "hooks": [
         {
           "type": "command",
           "command": "/Users/roasbeef/.claude/hooks/pretooluse/my_custom_hook.sh"
         }
       ]
     }
   ]
   ```

### Hook Execution Order

Multiple hooks run in the order they appear in settings.json:
```json
"hooks": [
  { "command": "hook1.sh" },  // Runs first
  { "command": "hook2.sh" },  // Runs second
  { "command": "hook3.sh" }   // Runs third
]
```

---

## Hook Recipes

### Recipe 1: Auto-format Go code after edits

**PostToolUse hook**:
```bash
#!/bin/bash
input=$(cat)
tool=$(echo "$input" | jq -r '.tool // empty')
file_path=$(echo "$input" | jq -r '.parameters.file_path // empty')

if [[ "$tool" == "Edit" || "$tool" == "Write" ]] && [[ "$file_path" =~ \.go$ ]]; then
    gofmt -w "$file_path"
    goimports -w "$file_path"
    echo "✅ Formatted: $file_path"
fi
exit 0
```

### Recipe 2: Prevent commits without tests

**PreToolUse hook** (blocks Bash tool running git commit):
```bash
#!/bin/bash
input=$(cat)
tool=$(echo "$input" | jq -r '.tool // empty')
command=$(echo "$input" | jq -r '.parameters.command // empty')

if [[ "$tool" == "Bash" ]] && [[ "$command" =~ git[[:space:]]commit ]]; then
    # Check if tests pass
    if ! go test ./... > /dev/null 2>&1; then
        echo "🚫 BLOCKED: Tests must pass before commit"
        exit 1
    fi
fi
exit 0
```

### Recipe 3: Log all Claude tool usage

**PostToolUse hook**:
```bash
#!/bin/bash
input=$(cat)
tool=$(echo "$input" | jq -r '.tool // empty')
timestamp=$(date '+%Y-%m-%d %H:%M:%S')

echo "$timestamp - Tool: $tool" >> ~/.claude/tool-usage.log
exit 0
```

---

## Troubleshooting

### Hook not running

1. **Check permissions**: `chmod +x hook-script.sh`
2. **Check settings.json**: Ensure hook is not commented out
3. **Check script errors**: Run hook manually to test
4. **Check logs**: Look for errors in Claude Code output

### Hook blocking unintentionally

1. **Check exit code**: Hook scripts should `exit 0` to allow continuation
2. **Add debug output**: `echo "Debug: hook running"` to see execution
3. **Test script directly**: `cat sample-input.json | ./hook-script.sh`

### Performance issues

If hooks are slowing down Claude:
1. **Disable expensive hooks** (like auto-test)
2. **Add timeouts** to hook scripts
3. **Make hooks async** where possible
4. **Cache results** in hooks

---

## Best Practices

1. **Keep hooks fast**: Hooks should complete in <1 second
2. **Make hooks idempotent**: Running multiple times shouldn't cause issues
3. **Provide feedback**: Use echo to show what hooks are doing
4. **Don't block unnecessarily**: Only exit 1 for critical issues
5. **Use environment variables**: For optional/configurable behavior
6. **Test hooks thoroughly**: Before enabling in production
7. **Document custom hooks**: Explain what they do and why

---

## Security Considerations

⚠️ **Important**: Hooks run with your shell environment and credentials.

- **Never run untrusted hooks**
- **Review hooks before enabling**
- **Be careful with hooks that modify files**
- **Don't commit sensitive data in hooks**
- **Use environment variables for secrets**

---

## Feedback & Customization

These hooks are designed for your Bitcoin/Lightning development workflow. Feel free to:

- **Disable hooks you don't need**
- **Customize keyword detection** in context_enhancer.py
- **Add project-specific hooks**
- **Create hooks for your specific workflows**

For hook ideas and examples, see: https://docs.claude.com/en/docs/claude-code/hooks-guide

---

## Summary of What Changed

### New Hook Directories
- `hooks/pretooluse/` - Hooks that run before tool execution
- `hooks/posttooluse/` - Hooks that run after tool execution
- `hooks/userpromptsubmit/` - Hooks that enhance prompts
- `hooks/sessionstart/` - Hooks that run at session start
- `hooks/sessionend/` - Hooks that run at session end
- `hooks/precompact/` - Hooks that run before compaction

### New Hooks Added
- **Context Enhancer**: Adds intelligent context to prompts
- **Sensitive File Guard**: Protects credentials from accidental edits
- **Task Workflow Guard**: Reminds about task management
- **Go Format Check**: Checks Go file formatting
- **Git Status Refresh**: Shows git changes after edits
- **Go Test Runner**: Optionally runs tests automatically
- **Coverage Tracker**: Optionally tracks test coverage
- **Session Context Loader**: Shows project state at startup
- **Session Context Saver**: Saves work for next session
- **Pre-Compact Archiver**: Preserves context before compaction

### Preserved Hooks
- All your existing notification hooks remain unchanged
- Ultrathink hook continues to work as before
