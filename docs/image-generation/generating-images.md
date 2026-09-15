# Generating images

Generate an image from a text description through the OpenRouter API.

If there is no description of what to generate, ask for one and stop.

## The script

`SCRIPT` below stands for `python3 <this directory>/scripts/openrouter-imagegen.py`, written out as
a literal absolute path. Resolve it once before running anything; if you do not know this
directory:

    find . ~/.claude -path '*image-generation/scripts/openrouter-imagegen.py' 2>/dev/null | head -1

No variables, nothing relative to a working directory.

## API key

`OPENROUTER_API_KEY` must be set in the environment. If it is not, tell the user to set it and
stop. Never attempt the request without it.

## Output path

- The user gave a location (a path or directory) — use it.
- The user gave a name but no directory — the current working directory, under that name.
- Neither — the current working directory, under a name made from the spec: 2-4 words, kebab-case,
  `.png` extension. E.g. "a golden coin with sparkles" becomes `golden-coin-sparkles.png`.

## Reference images

If the user pointed at existing image files for style or content reference, collect their paths.
They go after the prompt as extra arguments.

## Transparent background

Use `--transparent` only when the user explicitly asked for a transparent or removed background.
Otherwise add nothing about backgrounds to the prompt.

With `--transparent`:

- Rewrite the prompt to ask for the subject "on a solid pure white #FFFFFF background". The
  matting depends on this exact phrasing. Never ask the model for a transparent background
  directly — it does not work reliably.
- Pillow is required. Check with `python3 -c "import PIL"`; if that fails, install it with
  `pip install -r <this directory>/scripts/requirements.txt`, written out as a literal absolute path.
- It doubles API cost and latency. Tell the user if that is not obvious from context.

## Generate

    SCRIPT [--transparent] <output_path> "<prompt>" [reference_images...]

- Keep the prompt on a single line.
- If the model returns text instead of an image, the script prints that text to stderr and exits
  non-zero. Relay the text to the user; do not retry blindly.

## Report

Tell the user the path the image was saved to.
