#!/Users/roasbeef/.claude/skills/nano-banana/.venv/bin/python
"""Generate multiple images from a prompts file using Gemini's image generation API.

The prompts file can be JSON or newline-delimited text.

JSON format:
    [
        {"prompt": "a sunset", "filename": "sunset.png"},
        {"prompt": "a forest", "filename": "forest.png", "aspect": "16:9"}
    ]

Text format (one prompt per line, auto-numbered output):
    a sunset over mountains
    a forest with fog
    a beach at dawn

Usage:
    python batch_generate.py prompts.json output_dir/
    python batch_generate.py prompts.txt output_dir/ --model pro
    python batch_generate.py prompts.json output_dir/ --parallel 3

Examples:
    python batch_generate.py slides.json ./images/
    python batch_generate.py ideas.txt ./generated/ --model pro --aspect 16:9
"""

import argparse
import concurrent.futures
import json
import os
import sys
from pathlib import Path
from typing import Any

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


def get_client():
    """Initialize the Gemini client with API key from environment."""
    api_key = os.environ.get("GOOGLE_API_KEY") or os.environ.get("GEMINI_API_KEY")
    if not api_key:
        print("Error: GOOGLE_API_KEY or GEMINI_API_KEY environment variable not set.", file=sys.stderr)
        sys.exit(1)
    return genai.Client(api_key=api_key)


def load_prompts(prompts_path: str, default_aspect: str = "1:1") -> list[dict[str, Any]]:
    """Load prompts from a JSON or text file.

    Args:
        prompts_path: Path to the prompts file.
        default_aspect: Default aspect ratio for prompts without one specified.

    Returns:
        List of prompt dictionaries with keys: prompt, filename, aspect.
    """
    path = Path(prompts_path)
    if not path.exists():
        raise FileNotFoundError(f"Prompts file not found: {prompts_path}")

    content = path.read_text().strip()

    # Try JSON first.
    if path.suffix.lower() == ".json" or content.startswith("["):
        try:
            data = json.loads(content)
            prompts = []
            for i, item in enumerate(data):
                if isinstance(item, str):
                    prompts.append({
                        "prompt": item,
                        "filename": f"image_{i + 1:03d}.png",
                        "aspect": default_aspect,
                    })
                else:
                    prompts.append({
                        "prompt": item.get("prompt", item.get("text", "")),
                        "filename": item.get("filename", f"image_{i + 1:03d}.png"),
                        "aspect": item.get("aspect", default_aspect),
                    })
            return prompts
        except json.JSONDecodeError:
            pass

    # Fall back to newline-delimited text.
    lines = [line.strip() for line in content.split("\n") if line.strip()]
    return [
        {
            "prompt": line,
            "filename": f"image_{i + 1:03d}.png",
            "aspect": default_aspect,
        }
        for i, line in enumerate(lines)
    ]


def generate_single(
    client: genai.Client,
    prompt: str,
    output_path: Path,
    model: str,
    aspect_ratio: str,
) -> tuple[str, bool, str]:
    """Generate a single image.

    Returns:
        Tuple of (filename, success, message).
    """
    try:
        config = types.GenerateContentConfig(
            response_modalities=["IMAGE"],
            image_config=types.ImageConfig(aspect_ratio=aspect_ratio),
        )

        response = client.models.generate_content(
            model=model,
            contents=[prompt],
            config=config,
        )

        for part in response.parts:
            if part.inline_data is not None:
                image = part.as_image()
                output_path.parent.mkdir(parents=True, exist_ok=True)
                image.save(str(output_path))
                return (str(output_path), True, "Success")

        return (str(output_path), False, "No image in response")

    except Exception as e:
        return (str(output_path), False, str(e))


def batch_generate(
    prompts_path: str,
    output_dir: str,
    model: str = "gemini-2.5-flash-image",
    default_aspect: str = "1:1",
    parallel: int = 1,
) -> dict[str, Any]:
    """Generate multiple images from a prompts file.

    Args:
        prompts_path: Path to the prompts file (JSON or text).
        output_dir: Directory to save generated images.
        model: Model to use for generation.
        default_aspect: Default aspect ratio for prompts without one specified.
        parallel: Number of parallel workers.

    Returns:
        Summary dict with success/failure counts and details.
    """
    client = get_client()
    prompts = load_prompts(prompts_path, default_aspect)
    output_path = Path(output_dir)
    output_path.mkdir(parents=True, exist_ok=True)

    results = {"total": len(prompts), "success": 0, "failed": 0, "details": []}

    if parallel <= 1:
        # Sequential processing.
        for i, item in enumerate(prompts):
            print(f"[{i + 1}/{len(prompts)}] Generating: {item['filename']}")
            filepath, success, message = generate_single(
                client,
                item["prompt"],
                output_path / item["filename"],
                model,
                item["aspect"],
            )
            if success:
                results["success"] += 1
                print(f"  ✓ Saved: {filepath}")
            else:
                results["failed"] += 1
                print(f"  ✗ Failed: {message}")
            results["details"].append({
                "filename": item["filename"],
                "success": success,
                "message": message,
            })
    else:
        # Parallel processing.
        with concurrent.futures.ThreadPoolExecutor(max_workers=parallel) as executor:
            futures = {}
            for item in prompts:
                future = executor.submit(
                    generate_single,
                    client,
                    item["prompt"],
                    output_path / item["filename"],
                    model,
                    item["aspect"],
                )
                futures[future] = item

            for i, future in enumerate(concurrent.futures.as_completed(futures)):
                item = futures[future]
                print(f"[{i + 1}/{len(prompts)}] Processing: {item['filename']}")
                filepath, success, message = future.result()
                if success:
                    results["success"] += 1
                    print(f"  ✓ Saved: {filepath}")
                else:
                    results["failed"] += 1
                    print(f"  ✗ Failed: {message}")
                results["details"].append({
                    "filename": item["filename"],
                    "success": success,
                    "message": message,
                })

    return results


def main():
    parser = argparse.ArgumentParser(
        description="Generate multiple images from a prompts file.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument("prompts", help="Path to prompts file (JSON or newline-delimited text)")
    parser.add_argument("output_dir", help="Directory to save generated images")
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
        help="Default aspect ratio for prompts without one specified. Default: 1:1",
    )
    parser.add_argument(
        "--parallel", "-p",
        type=int,
        default=1,
        help="Number of parallel workers. Default: 1 (sequential)",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Output results as JSON",
    )

    args = parser.parse_args()

    # Resolve model name.
    model = MODELS.get(args.model, args.model)

    try:
        results = batch_generate(
            prompts_path=args.prompts,
            output_dir=args.output_dir,
            model=model,
            default_aspect=args.aspect,
            parallel=args.parallel,
        )

        print()
        if args.json:
            print(json.dumps(results, indent=2))
        else:
            print(f"Batch complete: {results['success']}/{results['total']} succeeded")
            if results["failed"] > 0:
                print(f"Failed: {results['failed']}")

        sys.exit(0 if results["failed"] == 0 else 1)

    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
