# Asana access

Every read and every write of Asana — tasks and documents alike — goes through
`scripts/asana.py`. Never reach Asana any other way.

## The access token

- The script reads an Asana personal access token from the `ASANA_ACCESS_TOKEN` environment
  variable.
- When it is not set, every command fails with `ASANA_ACCESS_TOKEN is not set`. Report that and
  stop. The user creates a personal access token in Asana's developer console and exports it in
  the environment the script runs in.
- Never ask for the token in the conversation, and never write it into a file or a command line.

## Running the script

`SCRIPT` below and in every Asana doc stands for `python3 <this directory>/scripts/asana.py`,
written out as a literal absolute path. Resolve it once before running anything; if you do not
know this directory:

    find . ~/.claude -path '*asana-tasks/scripts/asana.py' 2>/dev/null | head -1

Every command uses that literal path, and so does any command you hand to someone else. No
variables, nothing relative to a working directory — the shell that runs it is not necessarily
yours. The script needs Python 3 and nothing else.

## Referring to things

- **Task** — its gid, such as `1204567890123456`, or its Asana URL.
- **Workspace** — its name, case-insensitive, or its gid. `SCRIPT workspaces` lists every
  workspace.
- **Project** — its name, case-insensitive, its gid, or its Asana URL. `SCRIPT projects
  [--workspace <workspace>]` lists the projects that are not archived. A name is looked up among
  those, in every workspace unless `--workspace` narrows it. A name shared by two projects fails;
  give the gid or `--workspace`.
- **Document** — its id, or its URL.

## Output

- Every command prints JSON to stdout, except `body` and `doc-content`, which print raw markdown.
- Every id is an Asana gid, printed as a string.

## Errors

- Every command prints one line to stderr and exits non-zero on failure. Asana's own rejections
  read `Asana API error: …`.
- The script waits out Asana's rate limit and retries by itself, so a command can pause.
- Never work around an error. Never fall back to calling Asana another way. Report it and stop.
- A `create` that fails after Asana accepted the task leaves the task behind. Check with a listing
  before creating it again.
