#!/Users/roasbeef/.claude/skills/nano-banana/.venv/bin/python
"""Edit an existing image using Gemini's image generation API.

Usage:
    python edit_image.py input.png "edit instructions" output.png
    python edit_image.py input.png "remove the background" output.png --model pro

Examples:
    python edit_image.py photo.png "remove red-eye from the person" fixed.png
    python edit_image.py logo.png "change the color scheme to blue and white" logo_blue.png
    python edit_image.py room.png "add a plant in the corner" room_plant.png --model pro
"""

import argparse
import base64
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


def get_client():
    """Initialize the Gemini client with API key from environment."""
    api_key = os.environ.get("GOOGLE_API_KEY") or os.environ.get("GEMINI_API_KEY")
    if not api_key:
        print("Error: GOOGLE_API_KEY or GEMINI_API_KEY environment variable not set.", file=sys.stderr)
        sys.exit(1)
    return genai.Client(api_key=api_key)


def load_image_as_part(image_path: str) -> types.Part:
    """Load an image file and return it as a Gemini Part."""
    path = Path(image_path)
    if not path.exists():
        raise FileNotFoundError(f"Image not found: {image_path}")

    # Determine MIME type.
    suffix = path.suffix.lower()
    mime_types = {
        ".png": "image/png",
        ".jpg": "image/jpeg",
        ".jpeg": "image/jpeg",
        ".gif": "image/gif",
        ".webp": "image/webp",
    }
    mime_type = mime_types.get(suffix, "image/png")

    # Read and encode the image.
    with open(path, "rb") as f:
        image_data = f.read()

    return types.Part.from_bytes(data=image_data, mime_type=mime_type)


def edit_image(
    input_path: str,
    instructions: str,
    output_path: str,
    model: str = "gemini-2.5-flash-image",
) -> str:
    """Edit an existing image based on text instructions.

    Args:
        input_path: Path to the input image.
        instructions: Text instructions describing the edit.
        output_path: Path where the edited image will be saved.
        model: Model to use for editing.

    Returns:
        Path to the saved edited image.

    Raises:
        RuntimeError: If no image is generated.
    """
    client = get_client()

    # Load the input image.
    image_part = load_image_as_part(input_path)

    # Construct the edit prompt.
    edit_prompt = f"Using the provided image, {instructions}"

    # Build generation config.
    config = types.GenerateContentConfig(
        response_modalities=["IMAGE"],
    )

    response = client.models.generate_content(
        model=model,
        contents=[image_part, edit_prompt],
        config=config,
    )

    # Extract and save the edited image.
    for part in response.parts:
        if part.inline_data is not None:
            image = part.as_image()
            output = Path(output_path)
            output.parent.mkdir(parents=True, exist_ok=True)
            image.save(str(output))
            print(f"Edited image saved to: {output}")
            return str(output)

    raise RuntimeError("No image generated. Check your instructions and try again.")


def main():
    parser = argparse.ArgumentParser(
        description="Edit an existing image using Gemini.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument("input", help="Path to the input image")
    parser.add_argument("instructions", help="Text instructions for the edit")
    parser.add_argument("output", help="Output file path for the edited image")
    parser.add_argument(
        "--model", "-m",
        choices=list(MODELS.keys()) + list(MODELS.values()),
        default="flash",
        help="Model to use: 'flash' (fast) or 'pro' (high quality). Default: flash",
    )

    args = parser.parse_args()

    # Resolve model name.
    model = MODELS.get(args.model, args.model)

    try:
        edit_image(
            input_path=args.input,
            instructions=args.instructions,
            output_path=args.output,
            model=model,
        )
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
