# Gemini Image Generation API Reference

## Models

### gemini-2.5-flash-image

- **Purpose**: Fast, efficient image generation
- **Best for**: High-volume, low-latency tasks, rapid prototyping
- **Resolution**: Up to 2K
- **Reference images**: Up to 3

### gemini-3-pro-image-preview

- **Purpose**: Professional asset production
- **Best for**: Final production assets, marketing materials, high-quality output
- **Resolution**: Up to 4K (1K, 2K, 4K options)
- **Reference images**: Up to 14
- **Features**: Advanced reasoning, Google Search grounding

## Generation Parameters

### response_modalities

Controls what types of content the model can return.

| Value | Description |
|-------|-------------|
| `["IMAGE"]` | Image only |
| `["TEXT", "IMAGE"]` | Text and image |

### aspect_ratio

Controls the aspect ratio of generated images.

| Value | Use Case |
|-------|----------|
| `1:1` | Square, social media posts, profile pictures |
| `16:9` | Widescreen, presentations, YouTube thumbnails |
| `9:16` | Portrait, mobile stories, TikTok |
| `21:9` | Ultrawide, cinematic |
| `4:3` | Traditional photo, slides |
| `3:4` | Portrait photo |

### image_size (Gemini 3 Pro only)

Controls the output resolution.

| Value | Resolution |
|-------|------------|
| `1K` | ~1024px on longest edge |
| `2K` | ~2048px on longest edge |
| `4K` | ~4096px on longest edge |

## Python SDK Usage

### Basic Generation

```python
from google import genai
from google.genai import types

client = genai.Client(api_key="YOUR_API_KEY")

response = client.models.generate_content(
    model="gemini-2.5-flash-image",
    contents=["A sunset over mountains"],
    config=types.GenerateContentConfig(
        response_modalities=["IMAGE"],
        aspect_ratio="16:9",
    ),
)

for part in response.parts:
    if part.inline_data:
        image = part.as_image()
        image.save("output.png")
```

### Image Editing

```python
# Load existing image as Part
with open("input.png", "rb") as f:
    image_data = f.read()

image_part = types.Part.from_bytes(
    data=image_data,
    mime_type="image/png",
)

response = client.models.generate_content(
    model="gemini-2.5-flash-image",
    contents=[image_part, "Remove the background"],
    config=types.GenerateContentConfig(
        response_modalities=["IMAGE"],
    ),
)
```

### Multi-turn Chat

```python
chat = client.chats.create(model="gemini-2.5-flash-image")

# First turn: generate
response1 = chat.send_message(
    "A cozy cabin in the woods",
    config=types.GenerateContentConfig(response_modalities=["IMAGE"]),
)

# Second turn: refine
response2 = chat.send_message(
    "Add snow falling",
    config=types.GenerateContentConfig(response_modalities=["IMAGE"]),
)
```

## Rate Limits

| Tier | Requests per minute | Images per day |
|------|---------------------|----------------|
| Free | 10 | 50 |
| Pay-as-you-go | 60 | 1,500 |
| Enterprise | Custom | Custom |

## Batch API

For large-scale generation with higher rate limits (up to 24-hour turnaround):

```python
from google.genai import types

# Create batch request
batch = client.batches.create(
    model="gemini-2.5-flash-image",
    requests=[
        types.BatchRequest(contents=["prompt 1"]),
        types.BatchRequest(contents=["prompt 2"]),
        # ...
    ],
)

# Check status
status = client.batches.get(batch.name)
```

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| `SAFETY_BLOCKED` | Content policy violation | Modify prompt |
| `RATE_LIMITED` | Too many requests | Implement backoff |
| `INVALID_ARGUMENT` | Bad parameter | Check API docs |
| `RESOURCE_EXHAUSTED` | Quota exceeded | Wait or upgrade |

## SynthID Watermarking

All generated images include invisible SynthID watermarks for AI content identification. This is automatic and cannot be disabled.

## Supported Languages

The models work optimally with prompts in:
- English (EN)
- German (DE)
- Spanish (ES)
- French (FR)
- Italian (IT)
- Portuguese (PT)
- Japanese (JA)
- Korean (KO)
- Chinese (ZH)
