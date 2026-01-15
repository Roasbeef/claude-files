# Image Editing Patterns

## Editing Instruction Templates

When editing images with Gemini, frame instructions clearly to get predictable results.

## Common Editing Operations

### Background Changes

**Remove background:**
```
Remove the background, leaving only the main subject on a transparent/white background
```

**Replace background:**
```
Replace the background with [new background], keeping the main subject unchanged
```

**Blur background:**
```
Blur the background while keeping the foreground subject sharp
```

### Color Adjustments

**Change specific colors:**
```
Change the [element] from [old color] to [new color], keeping everything else the same
```

**Adjust overall color:**
```
Make the image more [warm/cool/vibrant/muted]
```

**Color scheme change:**
```
Adjust the color palette to [target scheme] while maintaining the composition
```

### Object Manipulation

**Add object:**
```
Add [object] in the [location], matching the existing style and lighting
```

**Remove object:**
```
Remove the [object] from the image, filling the space naturally
```

**Replace object:**
```
Replace the [existing object] with [new object], keeping the same position and scale
```

**Move object:**
```
Move the [object] from [current location] to [new location]
```

### Style Transfer

**Apply art style:**
```
Transform this image into [style] style while preserving the composition and subjects
```

**Change medium:**
```
Convert this photo into a [watercolor/sketch/oil painting] while keeping the subject recognizable
```

### Enhancement

**Improve quality:**
```
Enhance the image quality, making it sharper and more detailed
```

**Fix lighting:**
```
Adjust the lighting to be more [even/dramatic/natural]
```

**Add detail:**
```
Add more detail and texture to the [specific area]
```

## Multi-turn Editing Workflows

### Iterative Refinement

```
Turn 1: "A modern office space"
Turn 2: "Add plants throughout the space"
Turn 3: "Make the lighting warmer"
Turn 4: "Add a person working at the desk"
Turn 5: "Change the wall color to light blue"
```

### Progressive Enhancement

```
Turn 1: [Base image generation]
Turn 2: "Increase the level of detail"
Turn 3: "Improve the lighting and shadows"
Turn 4: "Add subtle texture to surfaces"
```

### A/B Testing Style

```
Session A:
Turn 1: "A tech startup logo"
Turn 2: "Make it more minimal"

Session B (new session):
Turn 1: "A tech startup logo"
Turn 2: "Make it more playful"
```

## Best Practices

### Be Specific

- Bad: "Make it better"
- Good: "Increase contrast and saturation in the foreground"

### Reference Original

- Bad: "Change the colors"
- Good: "Keep the composition but change the blue elements to green"

### One Change at a Time

For complex edits, break into steps:
```
Turn 1: "Change the background to a sunset"
Turn 2: "Add warm lighting reflections on the subject"
Turn 3: "Adjust the subject's colors to match the new lighting"
```

### Preserve Intent

When the original has specific elements you want to keep:
```
"Modify [specific element] while preserving [elements to keep]"
```

## Common Issues and Solutions

### Unwanted Changes

**Problem**: Model changes more than requested

**Solution**: Be more specific about what to preserve
```
"Change ONLY the [element], keeping the [list elements to preserve] exactly the same"
```

### Style Drift

**Problem**: Style changes across turns

**Solution**: Reinforce style in each prompt
```
"Continuing in the same [style] as before, add..."
```

### Loss of Detail

**Problem**: Details disappear in edits

**Solution**: Explicitly mention preserving details
```
"Add [element] while maintaining all existing details and textures"
```

## Slide-Specific Editing

### Adjusting for Text Space

```
"Shift the main visual to the [left/right/top/bottom] to create space for text on the [opposite side]"
```

### Color Consistency

```
"Adjust the colors to match the [brand color] palette while keeping the composition"
```

### Simplification

```
"Simplify the background, removing distracting elements, suitable for a presentation slide"
```

### Adding Visual Elements

```
"Add subtle [icons/shapes/patterns] that represent [concept] without overwhelming the main visual"
```
