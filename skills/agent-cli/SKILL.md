---
name: agent-cli
description: "Design and review CLIs for AI agent consumption. Covers machine-readable output, input hardening against hallucinations, schema introspection, context window discipline, dry-run safety rails, and skill file packaging. Use when building new CLIs, adding agent support to existing CLIs, reviewing CLI designs for agent compatibility, or wrapping APIs as CLI tools. Triggers: agent CLI, CLI for agents, machine-readable CLI, agent-first CLI, CLI agent DX."
---

# Agent-First CLI Design

Design CLIs where AI agents are first-class consumers alongside humans. Human DX optimizes for discoverability and forgiveness. Agent DX optimizes for predictability and defense-in-depth. Both can coexist in the same binary, but agent ergonomics must be intentionally designed — they do not emerge from human-first defaults.

## When to Use

- Building a new CLI that agents will invoke
- Adding agent support to an existing human-first CLI
- Reviewing CLI designs for agent compatibility
- Wrapping REST/gRPC APIs as CLI tools
- Writing CONTEXT.md, AGENTS.md, or skill files for a CLI

## When NOT to Use

- Pure library/SDK design (no CLI surface)
- GUI or TUI-only applications
- Internal scripts not exposed to agents

## Core Principle

**The agent is not a trusted operator.** Treat agent-generated CLI input with the same suspicion as untrusted user input to a web API. Agents hallucinate with high confidence — they generate plausible but wrong paths, IDs, and parameters. The CLI must be the last line of defense.

## Multi-Surface Architecture

A well-designed agent-first CLI serves multiple interfaces from a single binary and source of truth (e.g., a Discovery Document or OpenAPI spec).

```mermaid
graph TD
    Source[Discovery Document / OpenAPI] --> Core[Core Binary Logic]
    Core --> Human[Human CLI (Terminal)]
    Core --> MCP[MCP Server (stdio)]
    Core --> Ext[Agent Extension (Native)]
    Core --> Env[Env Vars (Headless)]
```

- **Human CLI:** Interactive, colorful, lenient flags (`--title "Doc"`).
- **MCP Server:** Typed JSON-RPC over stdio, auto-generated tools from schema.
- **Agent Extension:** Native capability installation (e.g., Gemini Extensions).
- **Env Vars:** Headless auth and config injection (`MYCLI_TOKEN`).

## Design Checklist

When building or reviewing an agent-facing CLI, verify each of the following areas. Address them in priority order — earlier items have higher impact per effort.

### 1. Machine-Readable Output

Machine-readable output is table stakes. Without it, agents must parse human-formatted tables and colored text, which is fragile and lossy.

**Requirements:**
- Support `--output json` (or `-o json`) on every command that produces output
- Set JSON as default when stdout is not a TTY (`if ! [ -t 1 ]; then output=json; fi`)
- Alternatively, respect `OUTPUT_FORMAT=json` environment variable
- Emit structured error objects on stderr, not prose messages

**Go implementation pattern:**
```go
type OutputFormat string

const (
    OutputHuman OutputFormat = "human"
    OutputJSON  OutputFormat = "json"
)

func detectFormat(cmd *cobra.Command) OutputFormat {
    if f, _ := cmd.Flags().GetString("output"); f == "json" {
        return OutputJSON
    }
    if os.Getenv("OUTPUT_FORMAT") == "json" {
        return OutputJSON
    }
    if !term.IsTerminal(int(os.Stdout.Fd())) {
        return OutputJSON
    }
    return OutputHuman
}
```

**Error output pattern:**
```json
{"error": {"code": "INVALID_RESOURCE_ID", "message": "resource ID contains query parameters", "field": "file-id", "value": "abc123?fields=name"}}
```

Avoid mixing human-readable and machine-readable output in the same stream. Diagnostics, progress bars, and warnings go to stderr. Structured results go to stdout.

### 2. Input Hardening Against Hallucinations

Agents hallucinate differently than humans typo. These are the failure modes to defend against, ranked by frequency:

| Input Type | Human Failure | Agent Failure | Defense |
|------------|---------------|---------------|---------|
| File paths | Misspelling | Path traversal (`../../.ssh/id_rsa`) | Canonicalize, sandbox to CWD |
| Resource IDs | Typos | Embedded query params (`id?fields=name`) | Reject `?`, `#`, `%` |
| Strings | Copy-paste garbage | Invisible control characters | Reject bytes below ASCII 0x20 |
| URLs/paths | Spaces in names | Pre-encoded strings (`%2e%2e`) | Reject `%` in IDs, encode at HTTP layer |
| JSON payloads | Syntax errors | Structurally valid but semantically wrong | Validate against schema |

**Go validation functions:**

```go
// validateResourceID rejects IDs that contain query parameters or encoding.
func validateResourceID(id string) error {
    for _, c := range id {
        switch {
        case c == '?' || c == '#':
            return fmt.Errorf("resource ID contains query character %q", c)
        case c == '%':
            return fmt.Errorf("resource ID contains percent-encoding")
        case c < 0x20:
            return fmt.Errorf("resource ID contains control character 0x%02x", c)
        }
    }
    return nil
}

// validateSafeOutputDir ensures the path resolves within the working directory.
func validateSafeOutputDir(path string) (string, error) {
    abs, err := filepath.Abs(path)
    if err != nil {
        return "", err
    }
    cwd, err := os.Getwd()
    if err != nil {
        return "", err
    }
    if !strings.HasPrefix(abs, cwd) {
        return "", fmt.Errorf("path %q escapes working directory", path)
    }
    return abs, nil
}
```

### 3. Raw JSON Payloads as First-Class Input

Humans hate writing nested JSON in the terminal. Agents prefer it.

A flag like `--title "My Doc"` makes ergonomic sense for a person but is lossy — it can’t express nested structures without creating layers of custom flag abstractions.

**Human-first** — 10 flags, flat namespace, can’t nest:
```bash
my-cli spreadsheet create \
  --title "Q1 Budget" --locale "en_US" --timezone "America/Denver" \
  --sheet-title "January" --sheet-type GRID \
  --frozen-rows 1 --frozen-cols 2 \
  --row-count 100 --col-count 10 --hidden false
```

**Agent-first** — one flag, the full API payload:
```bash
my-cli sheets spreadsheets create --json '{
  "properties": {"title": "Q1 Budget", "locale": "en_US", "timeZone": "America/Denver"},
  "sheets": [{"properties": {"title": "January", "sheetType": "GRID",
    "gridProperties": {"frozenRowCount": 1, "frozenColumnCount": 2, "rowCount": 100, "columnCount": 10},
    "hidden": false}}]
}'
```

**Why this matters:**
- The JSON version maps directly to the API schema.
- Zero translation loss between LLM generation and API input.
- Trivially generated by an LLM that knows the API schema.

**Design pattern:**
- Support `--json` (or `--params`) for all inputs, accepting the full API payload as-is.
- No custom argument layers between the agent and the API.
- Maintain convenience flags (`--title`) for humans, but treat `--json` as the primary path for agents.

**Priority rules:**
1. `--json` takes full precedence — ignore all other flags.
2. Log a warning if flags are also present (agent confusion signal).
3. Validate the JSON payload against the API schema before sending.

### 4. Schema Introspection at Runtime

Static documentation baked into prompts is expensive in tokens and goes stale. Make the CLI itself the documentation, queryable at runtime.

**Pattern: Discovery Documents as Source of Truth**
Use schema introspection (like Google's Discovery Document format) with dynamic `$ref` resolution. This lets the CLI become the canonical source for what the API accepts *right now*.

**Minimum viable introspection:**
```bash
mycli describe resource-create    # Dump params, types, required fields as JSON
mycli schema <method>             # Full method signature with request/response types
mycli --help --json               # Machine-readable help output
```

**Output format for `describe`:**
```json
{
  "command": "resource-create",
  "params": [
    {"name": "name", "type": "string", "required": true, "description": "Resource name"},
    {"name": "config", "type": "object", "required": false, "description": "Configuration blob"}
  ],
  "request_body": {"$ref": "#/definitions/CreateResourceRequest"},
  "response": {"$ref": "#/definitions/Resource"}
}
```

For API-backed CLIs, generate this from OpenAPI specs or protobuf descriptors. One source of truth, multiple interfaces.

### 5. Context Window Discipline

APIs return massive blobs. Agents pay per token and lose reasoning capacity with every irrelevant field.

**Field masks — LIMIT RESPONSE SIZE:**
Always use field masks to prune the output to exactly what the agent needs.
```bash
mycli resource list --fields "id,name,status"
mycli resource get <id> --fields "id,config.timeout"
```

**Pagination — STREAM, DON'T BUFFER:**
- **NDJSON Default:** When `--page-all` is used, emit one JSON object per page (or per item) on a new line.
- **Streaming:** The agent can process the stream line-by-line without buffering a massive top-level array into memory.
- **Example:**
```json
{"items": [{"id": 1}, {"id": 2}], "nextPageToken": "abc"}
{"items": [{"id": 3}, {"id": 4}], "nextPageToken": "def"}
```

**Large Content:**
- Truncate strings >1KB by default.
- For file downloads, write to disk (`--output-file`) instead of dumping binary to stdout.

### 6. Dry-Run and Safety Rails

Agents are fast, confident, and wrong. Mutating operations need a safety net.

**`--dry-run` requirements:**
- Validate all inputs and auth locally
- Show exactly what would be sent to the API (method, URL, body)
- Return a structured preview, not prose
- Exit 0 on valid dry-run, non-zero on validation failure

```json
{"dry_run": true, "method": "POST", "url": "/api/v1/resources", "body": {"name": "test"}, "validation": "passed"}
```

**Response Sanitization (Defense-in-Depth):**
Prompt injection can arrive via API responses (e.g., a malicious email body saying "Ignore previous instructions").
- Consider a `--sanitize <TEMPLATE>` flag to strip unsafe content before returning it to the agent.
- Integrate with safety filters (like Google Cloud Model Armor) to redact PII or harmful content in the CLI output stream.

**Confirmation for destructive operations:**
- Default to `--dry-run` for delete/destroy commands when `AGENT_MODE=true`
- Require explicit `--confirm` or `--force` to execute destructive operations
- Never prompt interactively — agents cannot respond to `y/n` prompts

### 7. Auth for Headless Environments

Agents cannot open browsers for OAuth redirects. Design auth for non-interactive use.

**Priority order:**
1. Environment variables: `MYCLI_TOKEN`, `MYCLI_API_KEY`
2. Credential files: `MYCLI_CREDENTIALS_FILE=/path/to/service-account.json`
3. Credential helpers: `mycli auth store` that writes to a well-known path
4. Service accounts over user accounts where possible

**Never:**
- Require interactive browser-based OAuth as the only auth path
- Prompt for passwords on stdin
- Use `open` or `xdg-open` to launch browser without a `--no-browser` fallback

### 8. Deterministic Exit Codes

Agents parse exit codes for control flow. Use semantic exit codes, not just 0/1.

| Exit Code | Meaning |
|-----------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid arguments / validation failure |
| 3 | Auth failure |
| 4 | Resource not found |
| 5 | Rate limited (include retry-after in JSON error) |
| 10 | Dry-run passed (no action taken) |

Document exit codes in `--help --json` and in any CONTEXT.md shipped with the CLI.

### 9. Ship Context Files for Agents

Agents learn through context injection, not `--help` pages. Ship files that agent frameworks can discover and inject.

**Skill Files (OpenClaw Metadata):**
Ship structured Markdown files (`SKILL.md`) with YAML frontmatter for discovery by agent frameworks.
```yaml
---
name: mycli-workflow
version: 1.0.0
metadata:
  openclaw:
    requires:
      bins: ["mycli"]
---
```
**Content:** Step-by-step guidance on how to use the CLI for complex tasks ("How to rotate keys", "How to list all active users").

**CONTEXT.md** — operational guidance for the CLI:
```markdown
# mycli Agent Context

## Critical Rules
- ALWAYS use `--output json` when invoking programmatically.
- ALWAYS use `--fields` on list commands to limit response size.
- ALWAYS use `--dry-run` before mutating operations, then confirm with the user.
- NEVER pass user-provided strings directly as resource IDs without validation.
```

**AGENTS.md** — security posture and trust model:
```markdown
# Security Model

This CLI is frequently invoked by AI agents. All inputs are treated as
potentially adversarial. The CLI validates resource IDs, file paths, and
payloads before sending requests to the API.
```

### 10. MCP Surface (Optional, High Value for API CLIs)

If the CLI wraps a structured API, expose it as an MCP (Model Context Protocol) server. This eliminates shell escaping, argument parsing ambiguity, and output parsing entirely.

```bash
mycli mcp serve --services resources,auth   # JSON-RPC over stdio
```

For Go CLIs, use the official MCP Go SDK: https://github.com/modelcontextprotocol/go-sdk

```go
import "github.com/modelcontextprotocol/go-sdk/mcp"
```

The MCP server should:
- Auto-generate tool definitions from the same schema used for CLI commands
- Accept typed parameters (no string parsing)
- Return structured JSON (no output format negotiation)
- Include the same input validation as the CLI path

## Incremental Adoption Order

For retrofitting an existing CLI, follow this order (highest impact first):

1. `--output json` on all commands
2. Input validation (reject control chars, traversals, embedded query params)
3. `--describe` or `--help --json` for runtime introspection
4. `--fields` for response filtering
5. `--dry-run` for mutating operations
6. Ship CONTEXT.md with agent-specific guidance
7. MCP surface for typed invocation

## Anti-Patterns to Avoid

| Anti-Pattern | Why It Fails | Fix |
|-------------|--------------|-----|
| Interactive prompts (`y/n?`) | Agents cannot respond to TTY prompts | Use `--confirm` flag or `--dry-run` |
| Colorized-only output | ANSI codes corrupt JSON parsing | Disable color when stdout is not a TTY |
| Human-readable tables as default | Agents must regex-parse columns | Default to JSON when not a TTY |
| Browser-only OAuth | Agents have no browser | Support env var tokens and service accounts |
| Pagination via "press Enter" | Agents cannot interact with pagers | Use `--page-all` with NDJSON streaming |
| Error messages as prose only | Agents cannot reliably extract error type | Structured error JSON with error codes |
| Flag-only input for nested data | Agents must discover and map N flags | Accept `--json` with full API payload |
| Undocumented side effects | Agents cannot intuit implicit behavior | Document all invariants in CONTEXT.md |
