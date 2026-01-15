#!/usr/bin/env python3
"""Create slide images from a structured outline or prompts file.

This script orchestrates slide image generation by calling the nano-banana
skill's generate_image.py script for each slide.

Usage:
    # From a prompts JSON file
    python create_slides.py prompts.json ./output/

    # From an outline JSON file
    python create_slides.py outline.json ./output/ --from-outline

    # With style preset
    python create_slides.py prompts.json ./output/ --style corporate

Examples:
    python create_slides.py blog_slides.json ./slides/
    python create_slides.py outline.json ./slides/ --from-outline --style minimalist
"""

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path
from typing import Any

# Path to nano-banana scripts.
NANO_BANANA_SCRIPTS = Path.home() / ".claude" / "skills" / "nano-banana" / "scripts"

# Style presets with prompt modifiers.
STYLE_PRESETS = {
    "corporate": {
        "suffix": "professional corporate style, clean lines, blue color scheme, minimal design",
        "colors": "blue and white",
    },
    "creative": {
        "suffix": "creative bold style, vibrant colors, dynamic composition, modern design",
        "colors": "vibrant multicolor",
    },
    "technical": {
        "suffix": "dark technical theme, circuit patterns, code aesthetic, tech presentation",
        "colors": "dark blue and cyan",
    },
    "minimalist": {
        "suffix": "minimalist design, lots of white space, subtle accents, clean typography",
        "colors": "black and white with accent",
    },
    "warm": {
        "suffix": "warm inviting style, earth tones, organic shapes, comfortable atmosphere",
        "colors": "warm earth tones",
    },
}


def load_prompts(prompts_path: str) -> list[dict[str, Any]]:
    """Load prompts from a JSON file."""
    with open(prompts_path) as f:
        data = json.load(f)

    prompts = []
    for i, item in enumerate(data):
        if isinstance(item, str):
            prompts.append({
                "prompt": item,
                "filename": f"slide_{i + 1:02d}.png",
            })
        else:
            prompts.append({
                "prompt": item.get("prompt", item.get("text", "")),
                "filename": item.get("filename", f"slide_{i + 1:02d}.png"),
                "aspect": item.get("aspect", "16:9"),
            })
    return prompts


def outline_to_prompts(outline_path: str, style: str | None = None) -> list[dict[str, Any]]:
    """Convert an outline JSON to image prompts.

    Outline format:
    {
        "title": "Presentation Title",
        "theme": "topic description",
        "slides": [
            {"type": "title", "title": "...", "subtitle": "..."},
            {"type": "content", "title": "...", "points": ["...", "..."]},
            {"type": "section", "title": "..."},
            {"type": "conclusion", "title": "...", "points": ["..."]}
        ]
    }
    """
    with open(outline_path) as f:
        outline = json.load(f)

    style_info = STYLE_PRESETS.get(style, STYLE_PRESETS["corporate"])
    theme = outline.get("theme", "professional presentation")

    prompts = []
    for i, slide in enumerate(outline.get("slides", [])):
        slide_type = slide.get("type", "content")
        title = slide.get("title", "")

        if slide_type == "title":
            subtitle = slide.get("subtitle", "")
            prompt = f"Title slide for {title}, {subtitle}, bold modern typography background, {style_info['suffix']}"
        elif slide_type == "section":
            prompt = f"Section divider slide for '{title}', abstract {theme} imagery, gradient {style_info['colors']}, full bleed background"
        elif slide_type == "conclusion":
            prompt = f"Conclusion slide background for '{title}', inspiring and uplifting, {style_info['suffix']}, space for summary points"
        else:  # content
            points = slide.get("points", [])
            concept = points[0] if points else title
            prompt = f"Slide background for '{title}': visual representation of {concept}, {style_info['suffix']}, clean composition with space for text on right"

        prompts.append({
            "prompt": prompt,
            "filename": f"slide_{i + 1:02d}_{slide_type}.png",
            "aspect": "16:9",
        })

    return prompts


def apply_style(prompts: list[dict[str, Any]], style: str) -> list[dict[str, Any]]:
    """Apply a style preset to existing prompts."""
    if style not in STYLE_PRESETS:
        return prompts

    style_info = STYLE_PRESETS[style]
    styled = []
    for p in prompts:
        new_prompt = f"{p['prompt']}, {style_info['suffix']}"
        styled.append({**p, "prompt": new_prompt})
    return styled


def generate_slide(
    prompt: str,
    output_path: Path,
    aspect: str = "16:9",
    model: str = "flash",
) -> tuple[bool, str]:
    """Generate a single slide image using nano-banana."""
    generate_script = NANO_BANANA_SCRIPTS / "generate_image.py"

    if not generate_script.exists():
        return False, f"nano-banana script not found: {generate_script}"

    # Call the script directly - it has an absolute shebang to its venv Python.
    cmd = [
        str(generate_script),
        prompt,
        str(output_path),
        "--aspect", aspect,
        "--model", model,
    ]

    try:
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=120,
        )
        if result.returncode == 0:
            return True, str(output_path)
        else:
            return False, result.stderr or "Unknown error"
    except subprocess.TimeoutExpired:
        return False, "Generation timed out"
    except Exception as e:
        return False, str(e)


def create_slides(
    prompts: list[dict[str, Any]],
    output_dir: str,
    model: str = "flash",
) -> dict[str, Any]:
    """Generate all slide images.

    Returns summary dict with success/failure counts.
    """
    output_path = Path(output_dir)
    output_path.mkdir(parents=True, exist_ok=True)

    results = {
        "total": len(prompts),
        "success": 0,
        "failed": 0,
        "slides": [],
    }

    for i, slide in enumerate(prompts):
        print(f"[{i + 1}/{len(prompts)}] Generating: {slide['filename']}")
        success, message = generate_slide(
            prompt=slide["prompt"],
            output_path=output_path / slide["filename"],
            aspect=slide.get("aspect", "16:9"),
            model=model,
        )

        if success:
            results["success"] += 1
            print(f"  ✓ Saved: {message}")
        else:
            results["failed"] += 1
            print(f"  ✗ Failed: {message}")

        results["slides"].append({
            "filename": slide["filename"],
            "prompt": slide["prompt"],
            "success": success,
            "message": message,
        })

    # Save prompts for reference.
    prompts_file = output_path / "slide_prompts.json"
    with open(prompts_file, "w") as f:
        json.dump(prompts, f, indent=2)
    print(f"\nPrompts saved to: {prompts_file}")

    return results


def main():
    parser = argparse.ArgumentParser(
        description="Create slide images from prompts or outline.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument("input", help="Path to prompts JSON or outline JSON file")
    parser.add_argument("output_dir", help="Directory to save slide images")
    parser.add_argument(
        "--from-outline",
        action="store_true",
        help="Input is an outline JSON (not raw prompts)",
    )
    parser.add_argument(
        "--style", "-s",
        choices=list(STYLE_PRESETS.keys()),
        help="Style preset to apply",
    )
    parser.add_argument(
        "--model", "-m",
        choices=["flash", "pro"],
        default="flash",
        help="Model to use: flash (default) or pro",
    )
    parser.add_argument(
        "--list-styles",
        action="store_true",
        help="List available style presets and exit",
    )

    args = parser.parse_args()

    if args.list_styles:
        print("Available style presets:\n")
        for name, info in STYLE_PRESETS.items():
            print(f"  {name}:")
            print(f"    {info['suffix']}")
            print()
        sys.exit(0)

    # Load prompts.
    try:
        if args.from_outline:
            prompts = outline_to_prompts(args.input, args.style)
        else:
            prompts = load_prompts(args.input)
            if args.style:
                prompts = apply_style(prompts, args.style)
    except FileNotFoundError:
        print(f"Error: File not found: {args.input}", file=sys.stderr)
        sys.exit(1)
    except json.JSONDecodeError as e:
        print(f"Error: Invalid JSON: {e}", file=sys.stderr)
        sys.exit(1)

    if not prompts:
        print("Error: No prompts found in input file", file=sys.stderr)
        sys.exit(1)

    print(f"Creating {len(prompts)} slides...")
    print(f"Output directory: {args.output_dir}")
    if args.style:
        print(f"Style: {args.style}")
    print()

    # Generate slides.
    results = create_slides(
        prompts=prompts,
        output_dir=args.output_dir,
        model=args.model,
    )

    print(f"\nComplete: {results['success']}/{results['total']} slides generated")
    if results["failed"] > 0:
        print(f"Failed: {results['failed']}")
        sys.exit(1)


if __name__ == "__main__":
    main()
