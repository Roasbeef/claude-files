# Slide Style Presets

Use these style presets to maintain visual consistency across slide decks.

## Available Presets

### Corporate

Professional business style suitable for formal presentations.

**Characteristics:**
- Clean, minimal design
- Blue color palette
- Structured layouts
- Professional typography feel

**Prompt suffix:**
```
professional corporate style, clean lines, blue color scheme, minimal design
```

**Best for:**
- Business reports
- Investor presentations
- Corporate communications
- Executive summaries

---

### Creative

Bold, dynamic style for engaging presentations.

**Characteristics:**
- Vibrant, bold colors
- Dynamic compositions
- Modern aesthetics
- High visual impact

**Prompt suffix:**
```
creative bold style, vibrant colors, dynamic composition, modern design
```

**Best for:**
- Marketing presentations
- Product launches
- Creative pitches
- Brand presentations

---

### Technical

Dark theme with tech aesthetics for technical content.

**Characteristics:**
- Dark backgrounds
- Circuit/code patterns
- Cyan and blue accents
- Developer-friendly aesthetic

**Prompt suffix:**
```
dark technical theme, circuit patterns, code aesthetic, tech presentation
```

**Best for:**
- Technical architecture
- Developer documentation
- API presentations
- Engineering updates

---

### Minimalist

Clean, spacious design with subtle accents.

**Characteristics:**
- Maximum white space
- Minimal elements
- Subtle color accents
- Focus on content

**Prompt suffix:**
```
minimalist design, lots of white space, subtle accents, clean typography
```

**Best for:**
- Thought leadership
- Executive briefings
- Simple overviews
- Elegant summaries

---

### Warm

Inviting, comfortable style with earth tones.

**Characteristics:**
- Earth tone palette
- Organic shapes
- Warm, welcoming feel
- Natural textures

**Prompt suffix:**
```
warm inviting style, earth tones, organic shapes, comfortable atmosphere
```

**Best for:**
- Team updates
- Culture presentations
- HR communications
- Community content

---

## Custom Style Creation

To create a custom style, define:

1. **Color scheme**: Primary, secondary, accent colors
2. **Mood keywords**: Professional, playful, serious, friendly
3. **Visual elements**: Geometric, organic, abstract, realistic
4. **Layout preference**: Centered, left-aligned, asymmetric

### Custom Style Template

```json
{
    "name": "my-custom-style",
    "suffix": "[mood] style, [visual elements], [color scheme], [layout]",
    "colors": "[primary and secondary colors]"
}
```

### Example: Startup Style

```json
{
    "name": "startup",
    "suffix": "modern startup style, geometric shapes, gradient purple and orange, bold and energetic",
    "colors": "purple and orange gradient"
}
```

## Applying Styles

### Via CLI

```bash
python create_slides.py prompts.json ./output/ --style corporate
```

### Via Manual Prompts

Append the style suffix to each prompt:
```
[your prompt], professional corporate style, clean lines, blue color scheme, minimal design
```

## Maintaining Consistency

For consistent slide decks:

1. **Use one style** throughout the entire deck
2. **Reference the same colors** in each prompt
3. **Keep aspect ratio consistent** (typically 16:9)
4. **Use similar composition** (e.g., always leave space on right)

## Style Mixing (Advanced)

Sometimes you may want to blend styles:

```
professional corporate style with warm earth tone accents
```

```
technical dark theme with minimalist layout
```

Keep mixing subtle to maintain visual coherence.
