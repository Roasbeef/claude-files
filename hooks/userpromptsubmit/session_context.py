#!/usr/bin/env python3
"""
UserPromptSubmit hook: Inject session context for continuity prompts.

This hook detects continuation-type prompts and automatically injects
the active session's TL;DR context to help Claude resume work after
compaction or at the start of a new conversation.
"""

import json
import sys
from pathlib import Path


def get_active_session() -> Path | None:
    """Find the active session file in .sessions/active/."""
    sessions_dir = Path(".sessions/active")
    if not sessions_dir.exists():
        return None

    sessions = list(sessions_dir.glob("*.md"))
    return sessions[0] if sessions else None


def extract_section(content: str, section_name: str, max_lines: int = 10) -> str:
    """Extract a markdown section by heading name."""
    lines = content.split('\n')
    result = []
    in_section = False

    for line in lines:
        if line.startswith(f"## {section_name}"):
            in_section = True
            continue
        if in_section:
            if line.startswith("## "):
                break
            result.append(line)
            if len(result) >= max_lines:
                break

    return '\n'.join(result).strip()


def get_session_context(session_file: Path) -> dict:
    """Extract key context from session file."""
    content = session_file.read_text()

    # Extract frontmatter
    shortname = ""
    compactions = 0
    for line in content.split('\n'):
        if line.startswith('shortname:'):
            shortname = line.split(':', 1)[1].strip()
        if line.startswith('compaction_count:'):
            try:
                compactions = int(line.split(':', 1)[1].strip())
            except ValueError:
                pass

    return {
        'shortname': shortname,
        'compactions': compactions,
        'tldr': extract_section(content, "TL;DR", 8),
        'next_steps': extract_section(content, "Next Steps", 5),
        'key_context': extract_key_context(content),
    }


def extract_key_context(content: str) -> str:
    """Extract the Key Context subsection."""
    lines = content.split('\n')
    result = []
    in_key_context = False

    for line in lines:
        if "**Key Context**" in line:
            in_key_context = True
            continue
        if in_key_context:
            if line.startswith("## ") or line.startswith("**") and "**:" in line:
                break
            if line.strip():
                result.append(line)
            if len(result) >= 8:
                break

    return '\n'.join(result).strip()


def is_continuation_prompt(prompt: str) -> bool:
    """Detect if this prompt is asking to continue previous work."""
    prompt_lower = prompt.lower().strip()

    # Direct continuation triggers
    triggers = [
        "continue",
        "resume",
        "keep going",
        "where were we",
        "what's next",
        "whats next",
        "what was i",
        "what were we",
        "pick up",
        "carry on",
        "let's continue",
        "lets continue",
        "go on",
        "proceed",
        "next step",
    ]

    for trigger in triggers:
        if trigger in prompt_lower:
            return True

    # Short prompts that imply continuation
    short_continuations = ["ok", "okay", "yes", "yep", "sure", "go", "next", "do it"]
    if prompt_lower in short_continuations:
        return True

    return False


def main():
    """Main hook entry point."""
    try:
        input_data = json.load(sys.stdin)
    except json.JSONDecodeError:
        # If we can't parse input, just pass through
        print("")
        sys.exit(0)

    prompt = input_data.get("prompt", "")

    # Check for active session
    session_file = get_active_session()
    if not session_file:
        # No active session, pass through unchanged
        print(prompt)
        sys.exit(0)

    # Check if this is a continuation prompt
    if not is_continuation_prompt(prompt):
        # Not a continuation, pass through unchanged
        print(prompt)
        sys.exit(0)

    # This is a continuation prompt - inject session context
    context = get_session_context(session_file)

    context_block = f"""[Session Context: {context['shortname']}]
{f"(After {context['compactions']} compaction(s))" if context['compactions'] > 0 else ""}

## TL;DR
{context['tldr']}

## Key Context
{context['key_context']}

## Next Steps
{context['next_steps']}

---
User request: {prompt}"""

    print(context_block)
    sys.exit(0)


if __name__ == "__main__":
    main()
