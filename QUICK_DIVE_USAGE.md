# Quick Dive vs Deep Dive - Usage Guide

## Command Comparison

| Aspect | `/quick-dive` (code-scout) | `/code-deep-dive` (architecture-archaeologist) |
|--------|---------------------------|-----------------------------------------------|
| **Time** | 2-3 minutes | 10-30+ minutes |
| **Scope** | Targeted, specific question | Comprehensive exploration |
| **Files Analyzed** | Max 10 files | Unlimited |
| **Output Size** | <1000 words | Extensive documentation |
| **Diagrams** | 0-1 simple diagrams | Multiple detailed diagrams |
| **Sub-agents** | None | Multiple parallel agents |
| **Best For** | Quick answers, specific queries | Full architectural understanding |

## When to Use Quick Dive

✅ **Perfect for:**
- "How does X work?"
- "Where is Y implemented?"
- "What does this function do?"
- Quick code navigation help
- Understanding specific flows
- Time-sensitive questions

❌ **Not suitable for:**
- Full system documentation
- Complex architectural analysis
- Multi-subsystem interactions
- Comprehensive security audits
- Complete dependency mapping

## Example Usage

### Quick Dive (Fast)
```
/quick-dive How does the payment validation work?
```
**Result**: 2-minute focused answer about payment validation logic

### Deep Dive (Comprehensive)
```
/code-deep-dive Analyze the entire payment processing architecture
```
**Result**: 20+ minute full analysis with multiple reports and diagrams

## Speed Optimization Techniques

The code-scout agent achieves speed through:

1. **No Ultrathink**: Skips deep planning phase
2. **Sequential Search**: No parallel sub-agents
3. **Smart Sampling**: Reads only most relevant sections
4. **Limited Recursion**: Max 2-3 levels deep
5. **Focused Scope**: Answers specific question only
6. **Minimal Diagrams**: Only if essential for clarity
7. **Time Boxing**: Hard 3-minute limit

## Escalation Path

The code-scout will automatically recommend using deep-dive when:
- Question requires >10 files
- Multiple subsystems involved
- Comprehensive documentation needed
- Full architectural understanding required

## Tips for Best Results

### For Quick Dive:
- Ask specific, focused questions
- Target single components or flows
- Use when you need answers fast
- Ideal for debugging sessions

### For Deep Dive:
- Use for documentation generation
- Full system understanding
- Architectural decisions
- Complete dependency analysis