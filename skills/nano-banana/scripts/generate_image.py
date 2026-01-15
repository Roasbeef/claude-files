#!/Users/roasbeef/.claude/skills/nano-banana/.venv/bin/python
"""Generate a single image from a text prompt using Gemini's image generation API.

Usage:
    python generate_image.py "prompt" output.png
    python generate_image.py "prompt" output.png --model gemini-3-pro-image-preview
    python generate_image.py "prompt" output.png --aspect 16:9 --size 4K

Examples:
    python generate_image.py "a sunset over mountains" sunset.png
    python generate_image.py "modern office interior" office.png --aspect 16:9
    python generate_image.py "product photo of headphones" product.png --model gemini-3-pro-image-preview --size 4K
"""

import argparse
import os
import sys
from pathlib import Path

try:
    from google import genai
    from google.genai import types
except ImportError:
    print("Error: google-genai package not installed.", file=sys.stderr)
    print("Install with: pip install google-genai", file=sys.stderr)
    sys.exit(1)


# Available models.
MODELS = {
    "flash": "gemini-2.5-flash-image",
    "pro": "gemini-3-pro-image-preview",
}

# Supported aspect ratios.
ASPECT_RATIOS = ["1:1", "16:9", "9:16", "21:9", "4:3", "3:4"]

# Supported resolutions (Gemini 3 Pro only).
RESOLUTIONS = ["1K", "2K", "4K"]


def get_client():
    """Initialize the Gemini client with API key from environment."""
    api_key = os.environ.get("GOOGLE_API_KEY") or os.environ.get("GEMINI_API_KEY")
    if not api_key:
        print("Error: GOOGLE_API_KEY or GEMINI_API_KEY environment variable not set.", file=sys.stderr)
        sys.exit(1)
    return genai.Client(api_key=api_key)


def generate_image(
    prompt: str,
    output_path: str,
    model: str = "gemini-2.5-flash-image",
    aspect_ratio: str = "1:1",
    resolution: str | None = None,
) -> str:
    """Generate an image from a text prompt.

    Args:
        prompt: Text description of the image to generate.
        output_path: Path where the generated image will be saved.
        model: Model to use for generation.
        aspect_ratio: Aspect ratio for the output image.
        resolution: Output resolution (1K, 2K, 4K). Only for Gemini 3 Pro.

    Returns:
        Path to the saved image.

    Raises:
        RuntimeError: If no image is generated.
    """
    client = get_client()

    # Build image config for aspect ratio and resolution.
    image_config_args = {"aspect_ratio": aspect_ratio}
    if resolution and "pro" in model.lower():
        image_config_args["image_size"] = resolution

    # Build generation config with image_config.
    config = types.GenerateContentConfig(
        response_modalities=["IMAGE"],
        image_config=types.ImageConfig(**image_config_args),
    )

    response = client.models.generate_content(
        model=model,
        contents=[prompt],
        config=config,
    )

    # Extract and save the image.
    for part in response.parts:
        if part.inline_data is not None:
            image = part.as_image()
            output = Path(output_path)
            output.parent.mkdir(parents=True, exist_ok=True)
            image.save(str(output))
            print(f"Image saved to: {output}")
            return str(output)

    raise RuntimeError("No image generated. Check your prompt and try again.")


def main():
    parser = argparse.ArgumentParser(
        description="Generate an image from a text prompt using Gemini.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument("prompt", help="Text description of the image to generate")
    parser.add_argument("output", help="Output file path (e.g., image.png)")
    parser.add_argument(
        "--model", "-m",
        choices=list(MODELS.keys()) + list(MODELS.values()),
        default="flash",
        help="Model to use: 'flash' (fast) or 'pro' (high quality). Default: flash",
    )
    parser.add_argument(
        "--aspect", "-a",
        choices=ASPECT_RATIOS,
        default="1:1",
        help="Aspect ratio. Default: 1:1",
    )
    parser.add_argument(
        "--size", "-s",
        choices=RESOLUTIONS,
        default=None,
        help="Output resolution (Gemini 3 Pro only). Options: 1K, 2K, 4K",
    )

    args = parser.parse_args()

    # Resolve model name.
    model = MODELS.get(args.model, args.model)

    # Validate resolution usage.
    if args.size and "pro" not in model.lower():
        print("Warning: --size is only supported for Gemini 3 Pro. Ignoring.", file=sys.stderr)
        args.size = None

    try:
        generate_image(
            prompt=args.prompt,
            output_path=args.output,
            model=model,
            aspect_ratio=args.aspect,
            resolution=args.size,
        )
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
