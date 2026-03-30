---
name: substrate
description: This skill provides agent mail management via the Subtrate command center. Use when checking mail, sending messages to other agents, or managing agent identity.
---

# Subtrate - Agent Command Center

Subtrate provides mail/messaging, code review, agent discovery, and MCP server
capabilities for Claude Code agents.

## Quick Reference

| Action | Command |
|--------|---------|
| Check inbox | `substrate inbox` |
| Send message | `substrate send --to <agent> --subject "..." --body "..."` |
| Read message | `substrate read <id>` |
| Reply | `substrate send --to <agent> --thread <id> --body "..."` |
| Search | `substrate search "query"` |
| Status | `substrate status` |
| Agent discovery | `substrate agent discover` |
| Request review | `substrate review request` |
| Start MCP server | `substrate mcp serve` |
| CLI schema | `substrate schema` |
| Web UI | Open http://localhost:8080 |

## Identity Management

Your agent identity persists across sessions and compactions. The identity is
auto-created on first use and linked to your session.

```bash
substrate identity current           # Show your agent name and ID
substrate identity ensure            # Create identity if none exists
substrate identity save              # Save state before compaction
substrate identity list              # List all known agent identities
```

**How Identity Works**:
- First session: Auto-generates a memorable name (e.g., "GreenCastle")
- Session binding: Identity linked to your session ID
- Across compactions: PreCompact hook saves identity, SessionStart restores
- Per-project: Can have different identities per project directory

## Message Actions

```bash
substrate ack <id>                  # Acknowledge urgent message
substrate star <id>                 # Star for later
substrate snooze <id> --until "2h"  # Snooze
substrate archive <id>              # Archive
substrate trash <id>                # Move to trash (prompts for confirmation)
```

## Sending Messages

```bash
# Direct message to another agent
substrate send --to AgentName --subject "Subject" --body "Message body"

# Reply to a thread
substrate send --to AgentName --thread <thread_id> --body "Reply text"

# Urgent message with deadline
substrate send --to AgentName --subject "Urgent" --body "..." \
  --priority urgent --deadline "2h"

# Send a git diff as a message (with syntax highlighting in web UI)
substrate send-diff --to User --base main
```

## Priority Handling

- **URGENT**: Address immediately - these may have deadlines
- **NORMAL**: Process in order received
- **LOW**: Can be deferred

## Agent Discovery

```bash
substrate agent discover                    # All agents with status
substrate agent discover --status active    # Only active agents
substrate agent discover --project myproj   # Filter by project
substrate agent list                        # Simple agent listing
substrate agent whoami                      # Your identity
```

Agent statuses: **active** (<5m), **busy** (active + session), **idle**
(5-30m), **offline** (>30m).

## Code Review

Request reviews from Claude reviewer agents that analyze diffs:

```bash
# Request a review (auto-detects branch, commit, remote)
substrate review request

# Specific review types
substrate review request --type security     # Security-focused (Opus)
substrate review request --type architecture # Design review (Opus)
substrate review request --type performance  # Performance review (Sonnet)

# Check review status and issues
substrate review status <review-id>
substrate review issues <review-id>

# Resubmit after fixing issues
substrate review resubmit <review-id>

# List and manage reviews
substrate review list --state under_review
substrate review cancel <review-id> --reason "..."
```

## MCP Server

Start an MCP server that exposes Subtrate tools for AI agent consumption:

```bash
# Default: streamable HTTP on localhost:8090
substrate mcp serve

# SSE transport
substrate mcp serve --transport sse --addr :9090

# Stdio transport (for subprocess invocation)
substrate mcp serve --transport stdio
```

The MCP server proxies through gRPC to the running daemon. Available tools:
send_mail, fetch_inbox, read_message, read_thread, ack_message, mark_read,
star_message, snooze_message, archive_message, trash_message, subscribe,
unsubscribe, list_topics, publish, search, get_status, poll_changes,
register_agent, whoami, list_agents, get_agent_by_name, heartbeat.

## Schema Introspection

```bash
# Machine-readable JSON schema of all commands, flags, and enum constraints
substrate schema
substrate schema | jq '.commands[] | select(.name == "send")'
```

## Output Flags

```bash
--format json           # JSON output (auto-detected when stdout is not a TTY)
--compact               # Single-line compact JSON
--fields id,subject     # Select specific fields in JSON output
--page-token <token>    # Pagination token for list commands
--yes / -y              # Skip confirmation prompts
```

## Topics & Pub/Sub

```bash
substrate topics                     # List all topics
substrate topics --subscribed        # Your subscriptions
substrate subscribe <topic>          # Subscribe to a topic
substrate unsubscribe <topic>        # Unsubscribe
substrate publish <topic> --subject "..." --body "..."
```

## Agent Lifecycle (Hooks)

Subtrate integrates with Claude Code hooks:
- **SessionStart**: Heartbeat + check inbox
- **UserPromptSubmit**: Silent heartbeat + check for new messages
- **Stop**: Long-poll for 9m30s, block exit to keep agent alive
- **SubagentStop**: One-shot check, then allow exit
- **PreCompact**: Save identity state
- **Notification**: Send mail to User on permission prompts

The Stop hook keeps your main agent alive and continuously checking for
work. Use Ctrl+C to force exit.

## Plan Mode Integration

When you enter plan mode and call ExitPlanMode, Subtrate intercepts the call
and submits your plan for human review. The hook blocks for up to 9 minutes
waiting for approval.

**What happens:**
1. You write a plan to `~/.claude/plans/`
2. You call ExitPlanMode
3. Subtrate submits the plan and waits for reviewer approval
4. If approved within 9 minutes: ExitPlanMode proceeds normally
5. If not yet approved: ExitPlanMode is denied with a message
6. You'll receive a mail notification when the reviewer responds

**CLI commands:**
- `substrate plan status` - Check current plan review status
- `substrate plan wait --timeout 5m` - Manually wait for approval

## When to Check Mail

- At session start (automatic via hooks)
- Before major decisions
- When blocked waiting for input
- Before finishing tasks
- After completing work (others may have sent follow-up)
