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
ai-config-manager -s DIR -t DIR [--dry-run] [--agents | --skills] [--filter NAME]
ai-config-manager -s DIR -t DIR -i
ai-config-manager -h | --help
```

### Flags

| Flag | Description |
| --- | --- |
| `-s`, `--source DIR` | Directory holding the `agents/` and `skills/` to install from. Required. |
| `-t`, `--target DIR` | Directory to install `agents/` and `skills/` into. Created if it does not exist. Required. |
| `--dry-run` | Print what would happen but write nothing. No short form. |
| `--agents` | Install only agents. Mutually exclusive with `--skills`. No short form. |
| `--skills` | Install only skills. Mutually exclusive with `--agents`. No short form. |
| `--filter NAME` | Install only the single item named `NAME`. Must be combined with `--agents` or `--skills`. No short form. |
| `-i` | Open an interactive terminal UI to choose what to install. See [Interactive mode](#interactive-mode) below. No long form. |
| `-h`, `--help` | Show help and exit 0. |

`-s`/`--source` and `-t`/`--target` are both accepted; Go's `flag` package
also accepts the single-dash long form (`-source`, `-target`). If a flag is
given more than once, the last occurrence wins.

`--agents` and `--skills` are mutually exclusive; passing both is a usage
error.

`--filter NAME` only makes sense alongside `--agents` or `--skills` — used
on its own, with neither, it's a usage error, since there would be no scope
to filter within. `NAME` is the bare item name, not a filename: an agent
stored as `agents/scout.md` is named `scout`, so `--filter scout.md` does
not match anything — the `.md` suffix is stripped before comparing, so
agents and skills are named the same way. Matching is exact and
case-sensitive: no prefixes, globs, or fuzzy matching. A `NAME` that matches
nothing is a fatal error (exit 1) rather than a run that quietly installs
nothing, since an exact name matching nothing is almost always a typo.

`-i` only combines with `-s`/`-t`. Combining it with `--dry-run`,
`--agents`, `--skills`, or `--filter` is a usage error — everything those
flags do is chosen inside the UI instead.

Exit codes: `0` on success, `1` on a usage error or a fatal error.

### Examples

```
ai-config-manager -s ./claude -t ~/.claude
ai-config-manager -s ./claude -t ~/.claude --dry-run
ai-config-manager -i -s ./claude -t ~/.claude
ai-config-manager -s ./claude -t ~/.claude --agents
ai-config-manager -s ./claude -t ~/.claude --skills --filter research
```

## Interactive mode

Passing `-i` opens a terminal UI for choosing what to install, instead of
installing everything the source ships. It needs a real interactive
terminal: if stdin is redirected or piped, it exits `1` with an error
rather than hanging waiting for input.

The UI renders inline, in the same terminal directly below where the
command was run — it does not take over the screen or use the alternate
screen buffer.

There are two tabs, **Agents** and **Skills**. Within each tab, items are
grouped under three headings:

- **to update** — present in the target but different from the source.
- **to install** — not present in the target yet.
- **up to date** — byte-identical to the target already. These are shown
  but cannot be selected, since installing an identical item would do
  nothing.

Selection is multi-select and shared across both tabs: agents and skills
chosen in either tab are installed together in one go when you confirm.
Each tab's label shows how many items are selected within it.

The list scrolls when it's longer than the terminal, with `↑ N more` /
`↓ N more` markers when there's more above or below the visible window.

A search bar filters the current tab by a case-insensitive substring match
on the item name. Each tab remembers its own search independently. This is
a browse aid, and not the same thing as `--filter` above — `--filter` is an
exact, case-sensitive match on one name that skips the UI entirely, while
the search bar just narrows what's visible while you pick. Items filtered
out of view stay selected.

Pressing enter opens a confirmation box listing everything currently
selected; the install only happens once you confirm it. After the install
finishes and the UI exits, a report is printed — see
[Interactive report](#interactive-report) below.

### Keys

| Key | Action |
| --- | --- |
| `j` / `k` (also ↓ / ↑) | Move the cursor down / up |
| `h` / `l` (also ← / →) | Previous / next tab |
| `space` | Select or deselect the item under the cursor |
| `enter` | Install the current selection (opens a confirmation first) |
| `/` | Open the search bar |
| `esc` | Close the search bar if open, else cancel the confirmation if open, else quit |
| `ctrl+c` | Quit at any time |

While the search bar is open, `j`, `k`, `h`, `l`, and `space` type into the
query instead of navigating — use the arrow keys to move the cursor while
searching. `left`/`right`, `backspace`, `delete`, `home`/`end`, and the
usual word-motions edit the query text. `enter` commits the search and
closes the bar without starting an install, so a stray enter while typing
can never trigger one. `esc` closes the search bar and clears the query.

### Interactive report

The interactive report differs from the [normal one](#output). Each
selected item is marked one of three statuses:

- **`installed`** — it was not present in the target before.
- **`updated`** — it was present in the target and different.
- **`failed`** — it could not be written, with the reason given.

Failed items are listed first, then installed, then updated, alphabetically
within each group. It ends with a summary line: `N installed, M updated,
K failed`.

Unlike a non-interactive run, which stops at the first write error, an
interactive install does not stop at the first failure — every selected
item is attempted and accounted for in the report. The command exits `1`
if any item failed.

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
