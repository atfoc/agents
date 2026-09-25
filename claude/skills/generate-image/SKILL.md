---
name: generate-image
description: Generates an image from a text description through the OpenRouter API with a bundled script, optionally using reference images or producing a transparent background. Use when you want an image, picture, icon, illustration or asset generated.
argument-hint: "[description of the image]"
---

# Generate an image

The subject is the description of what to generate: `$ARGUMENTS`, or a description earlier in the conversation. If there is none, ask for one and stop.

`SCRIPT` below stands for `python3 ${CLAUDE_SKILL_DIR}/scripts/openrouter-imagegen.py`, written out as a literal absolute path. Run the script; do not read it for information.

## 1. Check the API key

`OPENROUTER_API_KEY` must be set in the environment. If it is not, tell the user to set it and stop. Never attempt the request without it.

## 2. Choose the output path

- The user gave a location (a path or directory) — use it.
- The user gave a name but no directory — the current working directory, under that name.
- Neither — the current working directory, under a name made from the description: 2-4 words, kebab-case, `.png` extension. E.g. "a golden coin with sparkles" becomes `golden-coin-sparkles.png`.

## 3. Collect reference images

If the user pointed at existing image files for style or content reference, collect their paths. They go after the prompt as extra arguments.

## 4. Handle a transparent background

Use `--transparent` only when the user explicitly asked for a transparent or removed background. Otherwise add nothing about backgrounds to the prompt.

With `--transparent`:

- Rewrite the prompt to ask for the subject "on a solid pure white #FFFFFF background". The matting depends on this exact phrasing. Never ask the model for a transparent background directly.
- Pillow is required. Check with `python3 -c "import PIL"`; if that fails, install it with `pip install -r ${CLAUDE_SKILL_DIR}/scripts/requirements.txt`, written out as a literal absolute path.
- It doubles API cost and latency. Tell the user if that is not obvious from context.

## 5. Generate

    SCRIPT [--transparent] <output_path> "<prompt>" [reference_images...]

- Keep the prompt on a single line.
- If the model returns text instead of an image, the script prints that text to stderr and exits non-zero. Relay the text to the user; do not retry blindly.

## 6. Report

Tell the user the path the image was saved to.

The skill is finished once the image is saved and its path reported, or once it has stopped on a missing description or API key. Stop there.
