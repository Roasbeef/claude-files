#!/usr/bin/env python3
"""
Enhanced UserPromptSubmit hook that adds intelligent context to prompts.

Features:
- Preserves ultrathink functionality (-u suffix)
- Detects task-related prompts and suggests using /task-next
- Detects security/review prompts and adds security context
- Detects Bitcoin/Lightning keywords and adds protocol context
- Detects test/coverage prompts and adds testing guidance
"""

import json
import sys
import re
import os

def main():
    input_data = json.load(sys.stdin)
    prompt = input_data.get("prompt", "")

    # Track what context we're adding
    context_additions = []

    # 1. Handle ultrathink mode (preserve existing functionality)
    ultrathink_mode = False
    if prompt.rstrip().endswith("-u"):
        prompt = prompt.rstrip()[:-2].rstrip()
        ultrathink_mode = True

    # 2. Detect task management opportunities
    task_keywords = ["implement", "fix", "add feature", "create", "build", "refactor"]
    if any(keyword in prompt.lower() for keyword in task_keywords):
        # Check if we're in a project with tasks
        if os.path.isdir(".tasks"):
            # Check if there's an active task
            active_tasks = 0
            if os.path.isdir(".tasks/active"):
                for root, dirs, files in os.walk(".tasks/active"):
                    for file in files:
                        if file.endswith(".md"):
                            with open(os.path.join(root, file), 'r') as f:
                                if "status: in_progress" in f.read():
                                    active_tasks += 1

            if active_tasks == 0:
                context_additions.append(
                    "Note: Consider using /task-next to select a task from your task system before starting work."
                )

    # 3. Detect security/review context needs
    security_keywords = [
        "security", "vulnerability", "exploit", "attack", "DoS", "race condition",
        "consensus", "validation", "mempool", "transaction", "reorg"
    ]
    if any(keyword in prompt.lower() for keyword in security_keywords):
        context_additions.append(
            "Security Context: Remember to consider: DoS vectors, race conditions, "
            "resource exhaustion, consensus implications, and attack surface."
        )

    # 4. Detect Bitcoin/Lightning protocol context needs
    bitcoin_keywords = [
        "bitcoin", "lightning", "BOLT", "BIP", "TRUC", "v3 transaction",
        "package relay", "RBF", "CPFP", "mempool", "consensus", "p2p"
    ]
    if any(keyword in prompt.lower() for keyword in bitcoin_keywords):
        context_additions.append(
            "Bitcoin/Lightning Context: Ensure protocol compliance (BIPs/BOLTs), "
            "consider re-org safety, verify fee calculations, and test edge cases."
        )

    # 5. Detect test coverage context needs
    test_keywords = ["test", "coverage", "testing", "unit test", "integration test", "fuzz"]
    if any(keyword in prompt.lower() for keyword in test_keywords):
        context_additions.append(
            "Testing Context: Aim for >85% coverage with meaningful tests. "
            "Consider property-based testing with rapid for invariants. "
            "Test error paths, edge cases, and concurrent scenarios."
        )

    # 6. Build the enhanced prompt
    enhanced_prompt = prompt

    if context_additions:
        enhanced_prompt += "\n\n" + "\n".join(context_additions)

    # 7. Add ultrathink if requested
    if ultrathink_mode:
        enhanced_prompt += "\n\nultrathink"

    print(enhanced_prompt)
    sys.exit(0)

if __name__ == "__main__":
    main()
