# ai-config-manager

A small Go CLI that installs agent and skill definitions from a source
directory into a target directory. It generalizes the repo's two flavor-
specific shell installers (`claude/install-claude-config.sh` and
`cursor/install-cursor-config.sh`) into a single flavor-agnostic binary.

## Layout

The tool copies two kinds of items from `<source>` into `<target>`:

```
<source>/agents/<name>.md   ->  <target>/agents/<name>.md
<source>/skills/<name>/     ->  <target>/skills/<name>/
```

- An **agent** is a single `.md` file directly inside `<source>/agents/`.
  Files not ending in `.md`, and directories, are ignored.
- A **skill** is a directory directly inside `<source>/skills/` (typically
  containing a `SKILL.md` plus any supporting files). Loose files directly
  inside `skills/` are ignored.

Unlike the shell scripts, this tool does not append `.claude` or `.cursor`
to the target — you pass the full target root yourself, e.g. `-t ~/.claude`.

## Build

```sh
cd ai-config-manager
go build -o ai-config-manager .
```

The resulting binary shares its name with the folder and is gitignored.

Run the unit tests with:

```sh
go test ./...
```

## Usage

```
ai-config-manager -s DIR -t DIR [--dry-run]
ai-config-manager -h | --help
```

### Flags

| Flag | Description |
| --- | --- |
| `-s`, `--source DIR` | Directory holding the `agents/` and `skills/` to install from. Required. |
| `-t`, `--target DIR` | Directory to install `agents/` and `skills/` into. Created if it does not exist. Required. |
| `--dry-run` | Print what would happen but write nothing. No short form. |
| `-h`, `--help` | Show help and exit 0. |

`-s`/`--source` and `-t`/`--target` are both accepted; Go's `flag` package
also accepts the single-dash long form (`-source`, `-target`). If a flag is
given more than once, the last occurrence wins.

Exit codes: `0` on success, `1` on a usage error or a fatal error.

## Output

```
$ ai-config-manager -s ./claude -t ~/.claude --dry-run
DRY RUN — nothing written

AGENTS
------
updated    scout.md
updated    thinker.md
unchanged  worker.md

SKILLS
------
updated    make-plan
unchanged  implement-plan
unchanged  research

3 updated, 3 unchanged
```

Each item is reported as exactly one of two statuses:

- **`updated`** — the item was written. This includes items that did not
  exist in the target before the run; there is no separate "installed"
  status.
- **`unchanged`** — the item is byte-identical to what's already in the
  target, so nothing was written. For an agent this means the file's bytes
  match. For a skill it means the whole directory tree matches: same set of
  paths, same contents. A skill is replaced as a whole unit, so a leftover
  file in the target's copy that the source doesn't have makes the skill
  `updated`, because that stale file gets removed.

Unchanged items are not rewritten at all, so repeat runs are cheap and
don't churn modification times.

Agents are always reported before skills. Within each section, all
`updated` items come before all `unchanged` items, and each group is
sorted by name. An empty section prints `(none)`.

The `DRY RUN` banner is the only difference between dry-run output and the
real thing — a dry run and the run it describes produce output that diffs
cleanly against each other.

## What it will and will not touch

This is the part to trust before pointing it at a real config directory.

- The tool only ever installs into `<target>/agents` and `<target>/skills`,
  which it creates if missing. Within those, it writes only the items that
  come from `<source>/agents` and `<source>/skills`, one item at a time.
- **Anything else already in the target is left alone.** Your own agents
  and skills that already live there, and anything else in the target
  directory — such as Claude Code's own `settings.json`, `projects/`, or
  `history.jsonl` — are never read, written, removed, or reported.
- There is no prune or uninstall mode. Removing an item you previously
  installed is something you do yourself.
- Symlinks in the source are resolved and their content is copied, so the
  target never ends up holding a link that dangles once the source moves.
  A symlink already sitting in the target is replaced by a real file or
  directory. A symlinked directory nested inside a source skill is refused
  with an error rather than followed.
- Writes go through a temporary file or staging directory followed by a
  rename, so an interrupted run cannot leave a half-written item where a
  good one used to be.
- The run stops at the first write error and exits `1`. Items already
  written before the failure stay written — the tool does not roll back.
  Re-running after fixing the cause is safe and will finish the job.

## Errors

The tool exits `1` with a fatal error in these cases:

- The source directory does not exist or is not a directory.
- The source contains neither an `agents/` nor a `skills/` subdirectory.
  (Having only one of the two is fine — that section is simply empty.)
- The source and target are the same directory, or one is nested inside
  the other, which would make the tool copy into its own input.

## Relationship to the shell installers

`claude/install-claude-config.sh` and `cursor/install-cursor-config.sh`
still exist and are not replaced by this tool. Each hardcodes its own
source and appends `.claude`/`.cursor` to the target. `ai-config-manager`
is a flavor-agnostic replacement for the same underlying copy logic: point
it at any `<source>` with `agents/`/`skills/` subdirectories and any
`<target>`, and it performs the same kind of install for either flavor (or
any other tree with the same shape).
