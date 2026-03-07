---
name: agentic-code-reasoner
description: This skill enables deep, execution-free code analysis using the "Semi-Formal Reasoning" methodology. Use it for complex debugging, patch verification, or subtle logic questions where standard inspection might miss edge cases. It requires the generation of a "Reasoning Certificate" verifying logic paths before delivering a conclusion.
---

# Agentic Code Reasoner

This skill implements the "Agentic Code Reasoning" framework, focusing on **semi-formal reasoning**. It bridges the gap between unstructured reasoning and fully formal verification by using structured reasoning templates that force explicit evidence for every claim.

## When to Use

- **Patch Equivalence:** Verifying if two code patches produce the same semantic outcomes without execution.
- **Fault Localization:** Identifying the exact lines of buggy code given a failing test description.
- **Code Question Answering:** Answering nuanced questions about project-specific logic, library semantics, and edge cases.
- **Pre-commit Review:** Performing deep semantic analysis of changes to prevent regressions.

## The Semi-Formal Protocol

To perform agentic code reasoning, follow this iterative protocol to generate a **Reasoning Certificate**.

### Phase 1: Context & Premise Extraction
- **Goal:** Establish a baseline of facts from the codebase.
- **Action:** List all relevant function signatures, variable types, constants, and imported library behaviors involved.
- **Verification:** Use `grep_search` and `read_file` to ensure every premise is grounded in actual code.
- **Format:**
    - `[Premise] Function <name> defined in <file> accepts <args>`
    - `[Premise] Global constant <MAX_RETRIES> is set to <value>`

### Phase 2: Symbolic Execution Trace
- **Goal:** Mentally simulate execution paths for specific scenarios.
- **Action:**
    - **Control Flow:** Trace the path step-by-step (e.g., "Enter if block").
    - **Data Flow:** Track the state of key variables (e.g., "x is now Tainted").
    - **Loop Analysis:** For loops, explicitly trace "Iteration 0", "Iteration 1", and "Iteration N" to catch boundary errors.
- **Micro-Experiments:** You MAY use `run_shell_command` to execute small, isolated scripts (e.g., `python3 -c ...`) to verify *language semantics* (e.g., regex behavior, float precision), but NEVER to run the project's own code or tests.
- **Interprocedurality:** If a function is called, you MUST read its definition and include its trace as a sub-step.
- **Format:**
    1. `[Trace Step 1] Entry point <function> called with <params>.`
    2. `[Data Flow] Variable <user_input> is untrusted.`
    3. `[Loop Analysis] Iteration 0: i=0, condition true. Iteration 1: ...`

### Phase 3: Property Verification & Divergence Analysis
- **Goal:** Prove or disprove the target property (e.g., "is there a bug?", "are these equivalent?").
- **Action:**
    - **For Bugs:** Identify divergence points and generate **Ranked Predictions** (e.g., "Suspect #1: line 45 (80% confidence)").
    - **For Equivalence:** Trace both versions and identify if the side effects or return values differ.
- **Self-Correction Loop:**
    - If a trace contradicts a premise (e.g., "Trace says X is null, Premise says X is safe"), STOP.
    - **Backtrack:** Re-verify the premise using `read_file` or `grep_search`.
    - **Refine:** Update the premise or the trace with the new finding before proceeding.

### Phase 4: Formal Conclusion
- **Goal:** Provide the final verdict based strictly on the certificate.
- **Action:** Summarize the findings, referencing specific Trace Steps or Claims from Phase 3.

## Workflow Patterns

Refer to the specialized templates in `references/reasoning_templates.md` for specific tasks:
- **Patch Equivalence Verification Template**
- **Fault Localization (Bug Hunter) Template**
- **Deep Code Q&A Template**

## Guidelines

- **No Execution:** This skill is for reasoning *without* running the code. Do not suggest running tests as a primary verification method.
- **Evidence-First:** Never make a claim without a `[Premise]` or `[Trace Step]` supporting it.
- **Deep Context:** If a trace hits a library function whose behavior is unknown, use `grep_search` or `web_fetch` to find the documentation or implementation.
