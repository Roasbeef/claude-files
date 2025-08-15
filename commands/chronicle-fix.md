---
description: Analyze the current debugging session and create a runbook for future reference
argument-hint: [issue-name]
---

Use the Task tool to launch the debug-chronicler agent to analyze this debugging session and create a structured runbook.

The agent should:
1. Analyze the entire conversation transcript up to this point
2. Identify the bug that was fixed and how it was resolved
3. Create a comprehensive debugging runbook following the structured template
4. Save the runbook to the `debug-handbook/` directory with an appropriate filename

Issue context: $ARGUMENTS

Task prompt for the debug-chronicler agent:
```
Analyze the conversation transcript to extract the debugging journey and resolution. Focus on:

1. The initial problem/symptoms reported
2. All diagnostic steps taken (including those that didn't work)
3. The root cause discovered
4. The fix that was implemented
5. How to verify the fix worked

Create a structured runbook that would help another developer facing the same issue resolve it quickly. The runbook should include:
- Clear symptom identification checklist
- Diagnostic flowchart with decision trees
- Step-by-step resolution paths
- Verification steps
- Prevention recommendations

Issue Name/Context: $ARGUMENTS

Save the runbook to: debug-handbook/[appropriate-name]_runbook.md

Make the runbook practical and actionable - someone should be able to follow it without needing to read the original transcript.
```

Remember to create the debug-handbook directory if it doesn't exist, and ensure the runbook filename clearly indicates the type of issue it addresses.