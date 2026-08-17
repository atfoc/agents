---
name: generate-image
description: Generate an image from a text description via the OpenRouter API. User-invoked only — the agent does not trigger this on its own.
disable-model-invocation: true
---

# Generate an image

The subject is whatever the user gave you: the skill argument, or a description earlier in the
conversation. If there is no description of what to generate, ask for one and stop.

## Step 1 — Check the API key

Confirm `OPENROUTER_API_KEY` is set in the environment. If it is not, tell the user to set it and
stop — do not attempt the request without it.

## Step 2 — Work out the output path

- If the user gave a location (a path or directory), use it.
- If they gave a name but no directory, save into the current working directory under that name.
- If they gave no location or name at all, save into the current working directory under a name
  you generate from the spec: 2-4 words, kebab-case, `.png` extension (e.g. a prompt about "a
  golden coin with sparkles" becomes `golden-coin-sparkles.png`).

## Step 3 — Check for reference images

If the user pointed at existing image files for style or content reference, collect their paths —
these get passed as extra arguments after the prompt.

## Step 4 — Decide on transparency

Only use `--transparent` if the user explicitly asked for a transparent or removed background. If
so:

- Rewrite the prompt to explicitly ask for the subject "on a solid pure white #FFFFFF background"
  — the matting algorithm depends on this exact phrasing. Asking for a transparent background
  directly does not work reliably; do not do that.
- This requires Pillow. Check whether it's importable (`python3 -c "import PIL"`); if not,
  install it first: `pip install -r scripts/requirements.txt`.
- This doubles API cost and latency — mention that to the user if it's not obvious from context.

Otherwise, do not add anything about backgrounds to the prompt.

## Step 5 — Generate

Run the bundled script:

```bash
python3 scripts/openrouter-imagegen.py [--transparent] <output_path> "<prompt>" [reference_images...]
```

Keep the prompt as a single line. If the model returns text instead of an image, the script
prints that text to stderr and exits non-zero — relay it to the user rather than retrying blindly.

## Step 6 — Report

Tell the user the path the image was saved to.
