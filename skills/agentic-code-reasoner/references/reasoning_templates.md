# Agentic Reasoning Templates

Use these templates to structure the "Reasoning Certificate" for specific tasks.

## 1. Patch Equivalence Verification Template
Use this to prove that two code versions are semantically identical.

### [Phase 1] Context & Premises
*   **Source Code:** <file_path>
*   **Version A (Original):** <diff_A>
*   **Version B (Modified):** <diff_B>
*   **[Premise A]** <ground_truth_fact_about_code_logic_A>
*   **[Premise B]** <ground_truth_fact_about_code_logic_B>

### [Phase 2] Execution Traces & Micro-Experiments
*   **Trace Scenario:** <describe_the_specific_input_or_state>
*   **[Micro-Experiment Log]:** (Optional) Output of `python3 -c ...` to verify language semantics (e.g., regex, float precision).
*   **Version A Trace:**
    1. <Step 1>
    2. <Step 2>
    ...
*   **Version B Trace:**
    1. <Step 1>
    2. <Step 2>
    ...

### [Phase 3] Divergence Analysis
*   **Divergence Point:** <The exact Trace Step where A and B logic differs>
*   **Side Effect Comparison:** <Do both versions perform the same DB calls/API requests/state updates?>
*   **Return Value Comparison:** <Do both versions return the same value for all inputs?>
*   **[Counterexample]** If NOT EQUIVALENT, provide the specific input that causes different behavior.
*   **[Equivalence Proof]** If EQUIVALENT, explain why no counterexample exists.

### [Phase 4] Verdict
*   **Conclusion:** [EQUIVALENT] / [NOT EQUIVALENT]

---

## 2. Fault Localization (Bug Hunter) Template
Use this to find the root cause of a failing test or bug report.

### [Phase 1] Failing Test Semantics
*   **Test Name/File:** <test_file>
*   **Expected Behavior:** <The specific invariant that the test checks>
*   **Actual Outcome:** <The error message or incorrect state>

### [Phase 2] Path Tracing
*   **Trace:** Follow the execution path starting from the test's entry point until the failure.
*   **[Loop Analysis]:** Explicitly trace Iteration 0, 1, and N if a loop is involved.
*   **Observation:** Note the value of variables and state at each critical junction.

### [Phase 3] Divergence Analysis
*   **Expected Value:** <Value the variable SHOULD have at Step N>
*   **Actual Value:** <Value the variable ACTUALLY has at Step N>
*   **Root Cause Line:** <file:line_number>
*   **[Reasoning]** <Detailed explanation of why the logic at the root cause line is incorrect>

### [Phase 4] Ranked Suspects
*   **Suspect #1:** <file:line> (Confidence: High/Medium/Low) - <Reason>
*   **Suspect #2:** <file:line> (Confidence: High/Medium/Low) - <Reason>

### [Phase 5] Conclusion
*   **Verdict:** [BUG FOUND] / [FALSE POSITIVE]

---

## 3. Deep Code Q&A Template
Use this for answering nuanced questions about codebase behavior.

### [Phase 1] Definitions & Premises
*   **Key Functions:** List function signatures and behaviors involved.
*   **Assumptions:** Note any external factors (config, DB state, network).

### [Phase 2] Control Flow & Data Flow
*   **Scenario:** <Normal / Edge Case / Error>
*   **Control Flow Trace:**
    1. <Step 1: Function Entry>
    2. <Step 2: Condition Check>
    3. <Loop Analysis: Iteration 0...N>
*   **Data Flow Analysis:**
    *   **Variable <X>:** <Source> -> <Transformation> -> <Sink>
    *   **Taint Check:** Is <X> properly sanitized before use?

### [Phase 3] Semantic Properties
*   **Invariant Check:** Does the code always satisfy property X?
*   **Alternative Hypothesis:** "Could this code fail if <X> happens?" -> [Counter-Trace]

### [Phase 4] Conclusion
*   **Verdict:** <Final answer to the question>
