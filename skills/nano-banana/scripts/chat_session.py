#!/Users/roasbeef/.claude/skills/nano-banana/.venv/bin/python
"""Multi-turn image editing session using Gemini's image generation API.

This script maintains conversation state for iterative image refinement.
Each message in the session can either generate a new image or refine
the previously generated one.

Session state is saved to a JSON file, allowing sessions to be paused
and resumed later.

Usage:
    # Start a new interactive session
    python chat_session.py

    # Start with an initial prompt
    python chat_session.py --initial "a cozy cabin in the woods"

    # Resume a previous session
    python chat_session.py --session-file my_session.json

    # Non-interactive: send a single message to an existing session
    python chat_session.py --session-file my_session.json --message "make it more colorful"

Examples:
    # Interactive session
    python chat_session.py
    > a futuristic city at night
    [Image generated: output_001.png]
    > add flying cars
    [Image generated: output_002.png]
    > make the sky more purple
    [Image generated: output_003.png]
    > quit

    # Scripted refinement
    python chat_session.py --session-file logo.json --message "change colors to blue"
"""

import argparse
import json
import os
import sys
from datetime import datetime
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


def get_client():
    """Initialize the Gemini client with API key from environment."""
    api_key = os.environ.get("GOOGLE_API_KEY") or os.environ.get("GEMINI_API_KEY")
    if not api_key:
        print("Error: GOOGLE_API_KEY or GEMINI_API_KEY environment variable not set.", file=sys.stderr)
        sys.exit(1)
    return genai.Client(api_key=api_key)


class ChatSession:
    """Manages a multi-turn image generation/editing session."""

    def __init__(
        self,
        session_file: str | None = None,
        model: str = "gemini-2.5-flash-image",
        output_dir: str = ".",
    ):
        self.model = model
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(parents=True, exist_ok=True)
        self.session_file = session_file
        self.client = get_client()

        # Session state.
        self.history: list[dict[str, Any]] = []
        self.image_count = 0
        self.last_image_path: str | None = None

        # Load existing session if provided.
        if session_file and Path(session_file).exists():
            self._load_session(session_file)

    def _load_session(self, path: str) -> None:
        """Load session state from a JSON file."""
        with open(path) as f:
            data = json.load(f)
        self.history = data.get("history", [])
        self.image_count = data.get("image_count", 0)
        self.last_image_path = data.get("last_image_path")
        self.model = data.get("model", self.model)
        print(f"Loaded session with {len(self.history)} messages")

    def _save_session(self) -> None:
        """Save session state to a JSON file."""
        if not self.session_file:
            return
        data = {
            "model": self.model,
            "history": self.history,
            "image_count": self.image_count,
            "last_image_path": self.last_image_path,
            "updated_at": datetime.now().isoformat(),
        }
        with open(self.session_file, "w") as f:
            json.dump(data, f, indent=2)

    def _load_image_as_part(self, image_path: str) -> types.Part:
        """Load an image file and return it as a Gemini Part."""
        path = Path(image_path)
        suffix = path.suffix.lower()
        mime_types = {
            ".png": "image/png",
            ".jpg": "image/jpeg",
            ".jpeg": "image/jpeg",
            ".gif": "image/gif",
            ".webp": "image/webp",
        }
        mime_type = mime_types.get(suffix, "image/png")
        with open(path, "rb") as f:
            image_data = f.read()
        return types.Part.from_bytes(data=image_data, mime_type=mime_type)

    def send_message(self, message: str) -> str | None:
        """Send a message and generate/refine an image.

        Args:
            message: The user's prompt or refinement instruction.

        Returns:
            Path to the generated image, or None if generation failed.
        """
        # Build the content list.
        contents = []

        # If we have a previous image, include it for refinement.
        if self.last_image_path and Path(self.last_image_path).exists():
            image_part = self._load_image_as_part(self.last_image_path)
            contents.append(image_part)
            # Frame as an edit request.
            prompt = f"Based on this image, {message}"
        else:
            prompt = message

        contents.append(prompt)

        # Generate config.
        config = types.GenerateContentConfig(
            response_modalities=["IMAGE"],
        )

        try:
            response = self.client.models.generate_content(
                model=self.model,
                contents=contents,
                config=config,
            )

            # Extract the image.
            for part in response.parts:
                if part.inline_data is not None:
                    self.image_count += 1
                    output_path = self.output_dir / f"output_{self.image_count:03d}.png"
                    image = part.as_image()
                    image.save(str(output_path))

                    # Update state.
                    self.last_image_path = str(output_path)
                    self.history.append({
                        "role": "user",
                        "message": message,
                        "image_path": str(output_path),
                        "timestamp": datetime.now().isoformat(),
                    })
                    self._save_session()

                    return str(output_path)

            print("Warning: No image in response", file=sys.stderr)
            return None

        except Exception as e:
            print(f"Error: {e}", file=sys.stderr)
            return None

    def run_interactive(self) -> None:
        """Run an interactive session."""
        print("Multi-turn image generation session")
        print(f"Model: {self.model}")
        print(f"Output directory: {self.output_dir}")
        print("Type 'quit' or 'exit' to end the session")
        print("Type 'history' to see message history")
        print("-" * 40)

        while True:
            try:
                message = input("\n> ").strip()
            except (EOFError, KeyboardInterrupt):
                print("\nSession ended")
                break

            if not message:
                continue

            if message.lower() in ("quit", "exit", "q"):
                print("Session ended")
                break

            if message.lower() == "history":
                if not self.history:
                    print("No messages yet")
                else:
                    for i, item in enumerate(self.history):
                        print(f"{i + 1}. {item['message']}")
                        print(f"   → {item['image_path']}")
                continue

            # Generate image.
            print("Generating...")
            result = self.send_message(message)
            if result:
                print(f"Image saved: {result}")
            else:
                print("Failed to generate image")


def main():
    parser = argparse.ArgumentParser(
        description="Multi-turn image editing session using Gemini.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument(
        "--model", "-m",
        choices=list(MODELS.keys()) + list(MODELS.values()),
        default="flash",
        help="Model to use: 'flash' (fast) or 'pro' (high quality). Default: flash",
    )
    parser.add_argument(
        "--session-file", "-s",
        help="Path to session state file (JSON). Creates new if doesn't exist.",
    )
    parser.add_argument(
        "--output-dir", "-o",
        default=".",
        help="Directory to save generated images. Default: current directory",
    )
    parser.add_argument(
        "--initial", "-i",
        help="Initial prompt to start the session with",
    )
    parser.add_argument(
        "--message",
        help="Send a single message (non-interactive mode)",
    )

    args = parser.parse_args()

    # Resolve model name.
    model = MODELS.get(args.model, args.model)

    # Create session.
    session = ChatSession(
        session_file=args.session_file,
        model=model,
        output_dir=args.output_dir,
    )

    # Handle initial prompt.
    if args.initial and not session.history:
        print(f"Initial prompt: {args.initial}")
        print("Generating...")
        result = session.send_message(args.initial)
        if result:
            print(f"Image saved: {result}")
        else:
            print("Failed to generate image")
            sys.exit(1)

    # Non-interactive mode: send single message.
    if args.message:
        print("Generating...")
        result = session.send_message(args.message)
        if result:
            print(f"Image saved: {result}")
            sys.exit(0)
        else:
            print("Failed to generate image")
            sys.exit(1)

    # Interactive mode.
    if not args.message:
        session.run_interactive()


if __name__ == "__main__":
    main()
