---
description: Launch an architectural deep dive investigation into a specific area of the codebase
argument-hint: <area to investigate or question about the codebase>
---

## Architectural Deep Dive Investigation

I'll launch the architecture-archaeologist agent to perform a comprehensive deep dive into the codebase to answer your question or investigate the specified area.

### Investigation Focus
$ARGUMENTS

### Approach
I'll use the specialized architecture-archaeologist agent which will:
1. Plan thoroughly before diving in
2. Deploy multiple parallel sub-agents to analyze different aspects
3. Generate detailed markdown reports with mermaid diagrams
4. Synthesize findings into a comprehensive final document
5. Create architecture diagrams, call graphs, and sequence diagrams

Please use the Task tool to launch the architecture-archaeologist agent with the following prompt:

---
Perform a comprehensive architectural deep dive investigation into the following area or question:

$ARGUMENTS

Your investigation should:
1. Start with thorough planning of your approach
2. Break down the investigation into multiple parallel tracks
3. Launch Task agents in parallel to investigate different aspects independently
4. Have each sub-agent create its own markdown report with mermaid diagrams
5. Keep all intermediate reports (do not delete them)
6. Synthesize all findings into a final comprehensive document
7. Include extensive mermaid diagrams showing:
   - Overall architecture
   - Call graphs and dependencies
   - Sequence diagrams for key flows
   - Component relationships
   - Data flow patterns

Focus on being exhaustive and visual. This is a deep archaeological excavation of the code, not a surface-level overview.

Generate a final document that completely answers the investigation focus with exceptional clarity and detail.