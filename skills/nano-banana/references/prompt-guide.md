# Image Generation Prompt Guide

## Prompt Structure

A well-structured prompt typically includes:

1. **Subject**: What is the main focus?
2. **Style**: What artistic style or medium?
3. **Composition**: How is it framed?
4. **Lighting**: What's the mood/atmosphere?
5. **Details**: Specific elements to include

## Prompt Templates

### General Purpose

```
[subject] in [style], [composition], [lighting/mood], [additional details]
```

Example:
```
A red fox in a forest, watercolor painting style, close-up portrait, soft morning light, autumn leaves in background
```

### Product Photography

```
[product] on [surface/background], [lighting setup], [camera angle], professional product photography
```

Example:
```
Wireless headphones on white marble surface, soft studio lighting, 45-degree angle, professional product photography, minimalist
```

### Architectural/Interior

```
[space type] with [key elements], [style], [lighting], [atmosphere]
```

Example:
```
Modern living room with floor-to-ceiling windows, Scandinavian design, natural daylight, cozy and minimal
```

### Presentation/Slide Graphics

```
[concept] as [visual metaphor], [style], [color scheme], clean background for presentation
```

Example:
```
Growth and scalability as ascending bar chart with rocket, flat design illustration, blue and orange color scheme, clean white background for presentation
```

## Style Keywords

### Art Styles

| Keyword | Effect |
|---------|--------|
| `watercolor` | Soft, flowing paint effect |
| `oil painting` | Rich textures, classic look |
| `digital art` | Clean, modern illustration |
| `pencil sketch` | Hand-drawn appearance |
| `3D render` | Photorealistic CGI |
| `flat design` | Minimal, geometric shapes |
| `isometric` | 3D-like flat perspective |
| `pixel art` | Retro game aesthetic |

### Photography Styles

| Keyword | Effect |
|---------|--------|
| `portrait photography` | Focus on subject |
| `product photography` | Commercial quality |
| `macro photography` | Extreme close-up |
| `aerial view` | Top-down perspective |
| `street photography` | Candid urban shots |
| `studio lighting` | Controlled, professional |

### Mood/Atmosphere

| Keyword | Effect |
|---------|--------|
| `dramatic lighting` | High contrast |
| `soft light` | Gentle, diffused |
| `golden hour` | Warm sunset tones |
| `moody` | Dark, atmospheric |
| `vibrant` | Bright, saturated |
| `muted colors` | Desaturated, calm |
| `high key` | Bright, airy |
| `low key` | Dark, mysterious |

## Color Specification

### Direct Color Names
```
blue sky, red car, golden sunset
```

### Color Schemes
```
monochromatic blue, complementary orange and blue, analogous warm tones
```

### Hex/Brand Colors (less reliable)
```
corporate blue (#0066CC), brand colors matching [company]
```

## Negative Prompting

While Gemini doesn't have explicit negative prompts, you can guide by exclusion:

```
A modern kitchen, clean and minimal, no clutter, without people
```

Or by contrast:
```
A realistic portrait, not cartoonish, avoiding exaggeration
```

## Common Pitfalls

### Too Vague
- Bad: "A nice picture"
- Good: "A serene mountain lake at dawn, photorealistic, mist rising from water"

### Contradictory
- Bad: "A dark, bright sunny room"
- Good: "A room with dramatic contrast between sunlight and shadows"

### Overly Complex
- Bad: "A picture with 50 different elements each doing different things..."
- Good: Focus on 3-5 key elements maximum

## Prompt Refinement Process

1. **Start simple**: Basic subject and style
2. **Evaluate output**: What's missing or wrong?
3. **Add specifics**: Address issues with targeted additions
4. **Iterate**: Refine until satisfied

Example progression:
```
v1: "A coffee shop"
v2: "A cozy coffee shop with exposed brick walls"
v3: "A cozy coffee shop with exposed brick walls, warm lighting, morning atmosphere"
v4: "A cozy coffee shop with exposed brick walls, warm pendant lighting, morning sun through large windows, a few customers, watercolor style"
```

## Presentation Slide Prompts

For slide graphics, optimize for:
- Clear focal point
- Uncluttered composition
- Space for text overlay
- Consistent style across deck

### Title Slide
```
[topic] concept visualization, bold and modern, [brand colors], centered composition with space for title text
```

### Data/Chart Slide
```
[data concept] as abstract visualization, infographic style, [colors], clean white background, professional presentation graphic
```

### Transition/Section Slide
```
Abstract representation of [theme], gradient [colors], minimal, elegant, full bleed background for section divider
```
