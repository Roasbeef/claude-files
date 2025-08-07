---
argument-hint: <path-to-docs> or <file1.md file2.md ...>
description: Verify documentation accuracy against codebase
allowed-tools: Task, Glob, LS, Read
---

# Documentation Verification Task

You need to verify markdown documentation against the actual codebase using the documentation-double-checker agent.

## Input
The user wants to verify: $ARGUMENTS

## Your Task

1. **Determine what to verify**:
   - If $ARGUMENTS is a directory path, find all .md files in it
   - If $ARGUMENTS contains .md files, use those specific files
   - Use Glob or LS to discover markdown files if needed

2. **Launch the documentation-double-checker agent**:
   Use the Task tool with subagent_type: "documentation-double-checker" and provide this prompt:

   ```
   Verify the following documentation files against the codebase:
   [List the discovered markdown files]
   
   Check for accuracy of:
   - File paths and line numbers
   - Function/class names and signatures
   - Code examples and snippets
   - Architectural descriptions
   - Mermaid diagrams
   - API documentation
   
   Generate:
   1. Individual verification reports for each document
   2. Corrected versions for any documents with errors
   3. A comprehensive summary report named documentation_verification_report.md
   ```

3. **Report the results** showing:
   - Number of files verified
   - Overall accuracy percentage
   - Which files were corrected (if any)
   - Location of the verification report

Focus on thoroughness - every documentation claim should be verified against the actual code.