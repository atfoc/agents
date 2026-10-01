---
name: generate-image
description: Generates an image from a text description through the OpenRouter API with a bundled script, optionally using reference images or producing a transparent background. Use when you want an image, picture, icon, illustration or asset generated.
---

# Generate an image

Invoke with `$generate-image <description of the image>`. The subject is the description of what to generate in the request, or a description earlier in the conversation. If there is none, ask for one and stop.

`SCRIPT` below stands for `python3 "<absolute skill directory>/scripts/openrouter-imagegen.py"`. Resolve [the bundled script](scripts/openrouter-imagegen.py) relative to the directory containing this loaded `SKILL.md`, then write that resolved path literally in the command. Resolve [requirements.txt](scripts/requirements.txt) from the same skill directory. Use quoted absolute bundled paths, independent of the current working directory; do not invent an environment variable for the skill directory. Run the script; do not read it for information.

## 1. Check the API key

`OPENROUTER_API_KEY` must be set in the environment. Check its presence without printing its value. If it is not, tell the user to set it and stop. Never attempt the request without it.

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
- Pillow is required. Check with `python3 -c "import PIL"`; if that fails, install it with `pip install -r "<absolute skill directory>/scripts/requirements.txt"`, using the resolved literal absolute requirements path.
- It doubles API cost and latency. Tell the user if that is not obvious from context.

## 5. Generate

    SCRIPT [--transparent] "<output_path>" "<prompt>" [reference_images...]

- Keep the prompt on a single line. Pass the output path, prompt, and each reference path as separate shell-quoted arguments so their text is not executed by the shell.
- If the model returns text instead of an image, the script prints that text to stderr and exits non-zero. Relay the text to the user; do not retry blindly.

## 6. Report

Tell the user the path the image was saved to.

The skill is finished once the image is saved and its path reported, or once it has stopped on a missing description or API key. Stop there.
