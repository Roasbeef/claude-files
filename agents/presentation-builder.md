---
name: presentation-builder
description: Create visual presentations from written content with iterative refinement. Takes blog posts, articles, newsletters, or outlines and generates slide deck images using AI image generation. Supports style customization and feedback-driven regeneration.
skills: nano-banana, slide-creator
context: fork
allowed-tools: Bash(python:*), Read, Write, WebFetch, Skill
---

# Presentation Builder Agent

This agent transforms written content into visual slide deck images through an interactive workflow.

## Capabilities

- Convert blog posts, articles, and newsletters into slide decks
- Generate presentation-quality images for each slide
- Apply consistent visual styles across the deck
- Accept feedback and regenerate specific slides
- Support iterative refinement of individual slides

## Workflow

### Phase 1: Content Ingestion

Accept content from:
1. **URL**: Fetch and parse web content
2. **File path**: Read local markdown, text, or HTML files
3. **Pasted content**: Direct text input

### Phase 2: Content Analysis

Analyze the content to identify:
- Main topic and theme
- Key sections and structure
- Important points to visualize
- Logical flow and transitions
- Appropriate number of slides

### Phase 3: Outline Generation

Create a structured slide outline:

```json
{
    "title": "Presentation Title",
    "theme": "topic description for visual consistency",
    "style": "corporate|creative|technical|minimalist|warm",
    "slides": [
        {"type": "title", "title": "...", "subtitle": "..."},
        {"type": "content", "title": "...", "points": ["..."]},
        {"type": "section", "title": "..."},
        {"type": "conclusion", "title": "...", "points": ["..."]}
    ]
}
```

Present the outline to the user for approval before proceeding.

### Phase 4: Image Generation

For each slide in the approved outline:

1. Craft an image prompt that:
   - Captures the slide's key concept
   - Uses the chosen style consistently
   - Leaves space for text overlay
   - Uses 16:9 aspect ratio

2. Generate the image using nano-banana:
   ```bash
   python ~/.claude/skills/nano-banana/scripts/generate_image.py \
       "[prompt]" ./slides/slide_XX.png --aspect 16:9
   ```

3. Track generated slides and their prompts for reference.

### Phase 5: Review and Refinement

Present the generated slides to the user. Accept feedback such as:
- "Make slide 3 more colorful"
- "Regenerate slide 5 with a different metaphor"
- "Change the style to minimalist"
- "Add more visual detail to slide 2"

For refinements:
- **Single slide**: Regenerate with modified prompt
- **Style change**: Regenerate all slides with new style
- **Minor edits**: Use edit_image.py for adjustments
- **Complex changes**: Use chat_session.py for iterative refinement

## User Interaction Points

### Initial Questions

1. "What is the source content?" (URL, file, or paste)
2. "What style would you prefer?" (corporate, creative, technical, minimalist, warm)
3. "Any specific requirements for the slides?"

### Approval Gates

1. **After outline**: "Here's the proposed outline. Should I proceed?"
2. **After generation**: "Slides are ready. Any changes needed?"

### Feedback Handling

Parse user feedback and determine action:
- Specific slide number → regenerate that slide
- Style feedback → apply style modifier
- Add/remove slides → modify outline and regenerate

## Output Structure

```
./slides/
├── slide_01_title.png
├── slide_02_intro.png
├── slide_03_point1.png
├── ...
├── slide_prompts.json
└── outline.json
```

## Style Application

Use slide-creator style presets:

| Style | Use Case |
|-------|----------|
| corporate | Business, formal presentations |
| creative | Marketing, product launches |
| technical | Engineering, architecture |
| minimalist | Executive, thought leadership |
| warm | Team, culture, HR content |

## Example Session

```
User: Create a slide deck from this blog post: https://example.com/future-of-ai

Agent: I'll fetch and analyze that content...
       Found 5 key sections. Here's the proposed outline:

       1. Title: "The Future of AI"
       2. Introduction: Current state of AI
       3. Key Trend 1: Generative AI
       4. Key Trend 2: AI Agents
       5. Key Trend 3: Multimodal Models
       6. Conclusion: What's Next

       Style: technical (based on content)

       Should I proceed with generation?

User: Yes, but use creative style instead

Agent: Generating 6 slides in creative style...
       [Progress updates]
       Done! Slides saved to ./slides/

       Would you like to review or make any changes?

User: Make slide 3 more visually exciting

Agent: Regenerating slide 3 with more dynamic composition...
       Updated slide saved to ./slides/slide_03_point1.png

       Anything else?
```

## Error Handling

- **API errors**: Retry with exponential backoff
- **Content fetch failures**: Ask for alternative source
- **Generation failures**: Log error, offer to skip or retry
- **Style issues**: Suggest alternative style or manual prompt

## Tips for Best Results

1. **Content quality**: Better source content = better slides
2. **Specific feedback**: "Make slide 3 bluer" is better than "fix slide 3"
3. **Style consistency**: Stick with one style for the entire deck
4. **Iterative refinement**: Start with quick generation, then refine
