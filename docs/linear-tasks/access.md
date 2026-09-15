# Linear access

Every read and every write of Linear — issues and documents alike — goes through
`scripts/linear.py`. Never reach Linear any other way.

## The API key

- The script reads a Linear personal API key from the `LINEAR_API_KEY` environment variable.
- When it is not set, every command fails with `LINEAR_API_KEY is not set`. Report that and stop.
  The user creates a key in Linear's settings under Security & access → Personal API keys and
  exports it in the environment the script runs in.
- Never ask for the key in the conversation, and never write it into a file or a command line.

## Running the script

`SCRIPT` below and in every Linear doc stands for `python3 <this directory>/scripts/linear.py`,
written out as a literal absolute path. Resolve it once before running anything; if you do not
know this directory:

    find . ~/.claude -path '*linear-tasks/scripts/linear.py' 2>/dev/null | head -1

Every command uses that literal path, and so does any command you hand to someone else. No
variables, nothing relative to a working directory — the shell that runs it is not necessarily
yours. The script needs Python 3 and nothing else.

## Referring to things

- **Task** — its issue identifier, such as `ENG-123`. Case does not matter.
- **Team** — its key, such as `ENG`. `SCRIPT teams` lists every team's key and name.
- **Project** — its name, case-insensitive, or its id. `SCRIPT projects --team <key>` lists a
  team's projects. A name shared by two projects fails; give the id.
- **Document** — its id, or its Linear URL.

## Output

- Every command prints JSON to stdout, except `body` and `doc-content`, which print raw markdown.

## Errors

- Every command prints one line to stderr and exits non-zero on failure. Linear's own rejections
  read `Linear API error: …`.
- Never work around an error. Never fall back to calling Linear another way. Report it and stop.
- A `create` that fails after Linear accepted the issue leaves the issue behind. Check with a
  listing before creating it again.
