---
description: Fast, targeted code analysis for quick answers (2-3 minutes max)
argument-hint: <specific question about the codebase>
---

## Quick Code Analysis

I'll use the code-scout agent to quickly analyze the codebase and answer your specific question. This is a time-boxed, targeted investigation (not a comprehensive deep dive).

### Your Question
$ARGUMENTS

### Approach
The code-scout agent will:
- Provide a direct answer within 2-3 minutes
- Focus only on relevant code sections
- Skip exhaustive analysis in favor of speed
- Generate a concise summary (under 1000 words)

Please use the Task tool to launch the code-scout agent with the following prompt:

---
Perform a QUICK, TARGETED analysis to answer this specific question:

$ARGUMENTS

IMPORTANT CONSTRAINTS:
1. Complete analysis within 3 minutes maximum
2. Focus ONLY on what's needed to answer the question
3. Examine maximum 10 files
4. Keep response under 1000 words
5. Include only essential information
6. Use efficient search (grep/glob) before extensive reading
7. Skip comprehensive documentation
8. NO sub-agents or parallel analysis
9. Provide direct file:line references

If the question requires deep analysis of >10 files or multiple subsystems, recommend using the 'code-deep-dive' command instead.

Answer the question directly and concisely. Speed and relevance over exhaustiveness.