#!/usr/bin/env python3

import argparse
import base64
import json
import math
import os
import sys
import urllib.request

MIME_TYPES = {
    ".png": "image/png",
    ".jpg": "image/jpeg",
    ".jpeg": "image/jpeg",
    ".gif": "image/gif",
    ".webp": "image/webp",
    ".bmp": "image/bmp",
    ".svg": "image/svg+xml",
}

BLACK_BG_EDIT_PROMPT = (
    "Change the white background to a solid pure #000000 black. "
    "Keep everything else exactly unchanged."
)

BG_DIST = math.sqrt(3 * 255 * 255)


def encode_reference(path):
    ext = os.path.splitext(path)[1].lower()
    mime = MIME_TYPES.get(ext)
    if mime is None:
        print(f"Warning: unknown image type '{ext}' for {path}, skipping", file=sys.stderr)
        return None
    with open(path, "rb") as f:
        data = base64.b64encode(f.read()).decode()
    return {
        "type": "image_url",
        "image_url": {"url": f"data:{mime};base64,{data}"},
    }


def build_content_from_paths(prompt, reference_files):
    content = []
    for path in reference_files:
        block = encode_reference(path)
        if block is not None:
            content.append(block)
    content.append({"type": "text", "text": prompt})
    return content


def build_content_with_inline_image(prompt, png_bytes):
    data = base64.b64encode(png_bytes).decode()
    return [
        {"type": "image_url", "image_url": {"url": f"data:image/png;base64,{data}"}},
        {"type": "text", "text": prompt},
    ]


def call_openrouter(api_key, content):
    body = json.dumps({
        "model": "google/gemini-3-pro-image-preview",
        "modalities": ["text", "image"],
        "messages": [{"role": "user", "content": content}],
    }).encode()

    req = urllib.request.Request(
        "https://openrouter.ai/api/v1/chat/completions",
        data=body,
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
        },
    )

    try:
        with urllib.request.urlopen(req) as resp:
            result = json.loads(resp.read())
    except urllib.error.HTTPError as e:
        error_body = e.read().decode()
        print(f"API error {e.code}: {error_body}", file=sys.stderr)
        sys.exit(1)

    message = result["choices"][0]["message"]

    image_blocks = []
    content = message.get("content")
    if isinstance(content, list):
        image_blocks.extend(b for b in content if b.get("type") == "image_url")
    for b in message.get("images", []):
        if b.get("type") == "image_url":
            image_blocks.append(b)

    if image_blocks:
        url = image_blocks[0]["image_url"]["url"]
        _, b64data = url.split(",", 1)
        return base64.b64decode(b64data)

    if isinstance(content, str) and content:
        print("No image in response. Model replied with text:", file=sys.stderr)
        print(content, file=sys.stderr)
    else:
        print("No image found in response.", file=sys.stderr)
        print(json.dumps(message, indent=2)[:2000], file=sys.stderr)
    sys.exit(1)


def difference_matte(white_png, black_png):
    try:
        from PIL import Image
    except ImportError:
        print(
            "Error: Pillow is required for --transparent. "
            "Install it from requirements.txt next to this script.",
            file=sys.stderr,
        )
        sys.exit(1)

    import io

    white_img = Image.open(io.BytesIO(white_png)).convert("RGB")
    black_img = Image.open(io.BytesIO(black_png)).convert("RGB")

    if white_img.size != black_img.size:
        print(
            f"Error: white-bg ({white_img.size}) and black-bg ({black_img.size}) "
            "images differ in size; cannot matte. The model may have re-cropped on edit.",
            file=sys.stderr,
        )
        sys.exit(1)

    white_px = white_img.load()
    black_px = black_img.load()
    w, h = white_img.size
    out = Image.new("RGBA", (w, h))
    out_px = out.load()

    for y in range(h):
        for x in range(w):
            rW, gW, bW = white_px[x, y]
            rB, gB, bB = black_px[x, y]
            dr = rW - rB
            dg = gW - gB
            db = bW - bB
            dist = math.sqrt(dr * dr + dg * dg + db * db)
            alpha = 1.0 - (dist / BG_DIST)
            if alpha < 0.0:
                alpha = 0.0
            elif alpha > 1.0:
                alpha = 1.0

            if alpha > 0.01:
                r = int(min(255, max(0, rB / alpha)))
                g = int(min(255, max(0, gB / alpha)))
                b = int(min(255, max(0, bB / alpha)))
            else:
                r = g = b = 0

            out_px[x, y] = (r, g, b, int(round(alpha * 255)))

    buf = io.BytesIO()
    out.save(buf, format="PNG")
    return buf.getvalue()


def main():
    parser = argparse.ArgumentParser(description="Generate an image via OpenRouter")
    parser.add_argument("output", help="Path to save the generated image")
    parser.add_argument("prompt", help="Image generation prompt")
    parser.add_argument("references", nargs="*", help="Reference image files to attach")
    parser.add_argument(
        "--transparent",
        action="store_true",
        help=(
            "Produce a transparent-background PNG via two-pass difference matting. "
            "The prompt MUST instruct the model to render on a pure white #FFFFFF background."
        ),
    )
    args = parser.parse_args()

    api_key = os.environ.get("OPENROUTER_API_KEY")
    if not api_key:
        print("Error: OPENROUTER_API_KEY environment variable is not set", file=sys.stderr)
        sys.exit(1)

    if args.transparent:
        white_png = call_openrouter(
            api_key, build_content_from_paths(args.prompt, args.references)
        )
        black_png = call_openrouter(
            api_key, build_content_with_inline_image(BLACK_BG_EDIT_PROMPT, white_png)
        )
        rgba_png = difference_matte(white_png, black_png)
        with open(args.output, "wb") as f:
            f.write(rgba_png)
        print(f"Transparent image saved to {args.output}")
        return

    image_bytes = call_openrouter(
        api_key, build_content_from_paths(args.prompt, args.references)
    )
    with open(args.output, "wb") as f:
        f.write(image_bytes)
    print(f"Image saved to {args.output}")


if __name__ == "__main__":
    main()
