# Claude Code Hooks Directory

This directory contains all the custom hooks for your Claude Code environment.

## Directory Structure

```
hooks/
├── ultrathink_hook.py           # Original ultrathink hook (preserved)
├── pretooluse/                   # Hooks that run before tool execution
│   ├── sensitive_file_guard.sh  # Protects credentials/secrets
│   ├── task_workflow_guard.sh   # Reminds about task management
│   └── go_format_check.sh       # Checks Go formatting
├── posttooluse/                  # Hooks that run after tool execution
│   ├── git_status_refresh.sh    # Shows git changes (ENABLED)
│   ├── go_test_runner.sh        # Auto-run tests (disabled, use CLAUDE_AUTO_TEST=1)
│   └── coverage_tracker.sh      # Track coverage (disabled, use CLAUDE_TRACK_COVERAGE=1)
├── userpromptsubmit/            # Hooks that enhance prompts
│   └── context_enhancer.py      # Adds intelligent context to prompts
├── sessionstart/                # Hooks at session start
│   └── load_project_context.sh  # Shows project state
├── sessionend/                  # Hooks at session end
│   └── save_session_context.sh  # Archives session work
└── precompact/                  # Hooks before conversation compaction
    └── save_important_context.sh # Preserves context
```

## Quick Reference

### Always Active Hooks

✅ **Sensitive File Guard** - Blocks edits to .env, keys, credentials
✅ **Task Workflow Guard** - Reminds about task management
✅ **Go Format Check** - Warns about formatting issues
✅ **Git Status Refresh** - Shows changes after edits
✅ **Context Enhancer** - Adds intelligent context to prompts
✅ **Session Context** - Loads/saves project state
✅ **Pre-Compact Archive** - Preserves context before compacting

### Session Management Hooks

These hooks power the session management system (see `../SESSIONS.md`):

| Hook | Event | Function |
|------|-------|----------|
| `sessionstart/load_project_context.sh` | Session start | Displays active session TL;DR, progress, blockers |
| `sessionend/save_session_context.sh` | Session end | Archives session work and state |
| `precompact/save_important_context.sh` | Before compaction | Auto-saves session, outputs key context for summary |
| `userpromptsubmit/context_enhancer.py` | User prompt | Detects "continue"/"resume" and injects session context |

**How it works:**
1. **On startup**: If an active session exists in `.sessions/active/`, the SessionStart hook displays the TL;DR and suggests `/session-resume`
2. **During work**: Claude logs progress/decisions using `/session-log` commands
3. **Before compaction**: The PreCompact hook auto-checkpoints the session and outputs key context that survives in the compaction summary
4. **After compaction**: User says "continue" → UserPromptSubmit hook injects session context → Claude uses `/session-resume` for full restoration

### Optional Hooks (Disabled by Default)

🔘 **Go Test Runner** - Enable with: `export CLAUDE_AUTO_TEST=1`
🔘 **Coverage Tracker** - Enable with: `export CLAUDE_TRACK_COVERAGE=1`

## Documentation

See `../HOOKS.md` for complete documentation including:
- Detailed descriptions of each hook
- How to enable/disable hooks
- How to create custom hooks
- Hook recipes and examples
- Troubleshooting guide

## Configuration

Hooks are configured in `../settings.json`.

To disable a hook, comment it out in settings.json:
```json
// {
//   "type": "command",
//   "command": "/path/to/hook.sh"
// }
```

## Testing Hooks

To test a hook manually:
```bash
# Create sample input
echo '{"tool": "Edit", "parameters": {"file_path": "test.go"}}' | ./pretooluse/go_format_check.sh
```

## Security Note

⚠️ Hooks run with your shell credentials. Review all hooks before use.
