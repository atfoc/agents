# ai-config-manager

A small Go CLI that installs agent and skill definitions from a source
directory into a target directory, and removes ones already installed
there.

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

The tool is flavor-agnostic and does not append `.claude` or `.cursor` to
the target — you pass the full target root yourself, e.g. `-t ~/.claude` or
`-t ~/.cursor`.

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
ai-config-manager -s DIR -t DIR [--dry-run] [--agents | --skills] [--install NAME]...
ai-config-manager -t DIR (--agents | --skills) --uninstall NAME... [--dry-run]
ai-config-manager -s DIR -t DIR -i
ai-config-manager -h | --help
```

### Flags

| Flag | Description |
| --- | --- |
| `-s`, `--source DIR` | Directory holding the `agents/` and `skills/` to install from. Required, except with `--uninstall`, which rejects it. |
| `-t`, `--target DIR` | Directory to install `agents/` and `skills/` into, and to remove them from. Created if it does not exist. Required. |
| `--dry-run` | Print what would happen but write nothing. No short form. |
| `--agents` | Restrict the run to agents. Mutually exclusive with `--skills`. No short form. |
| `--skills` | Restrict the run to skills. Mutually exclusive with `--agents`. No short form. |
| `--install NAME` | Install only the item named `NAME`. Repeatable. Must be combined with `--agents` or `--skills`. No short form. |
| `--uninstall NAME` | Remove the item named `NAME` from the target. Repeatable. Must be combined with `--agents` or `--skills`. Cannot be combined with `-s` or `--install`. No short form. |
| `-i` | Open an interactive terminal UI to choose what to install and remove. See [Interactive mode](#interactive-mode) below. No long form. |
| `-h`, `--help` | Show help and exit 0. |

`-s`/`--source` and `-t`/`--target` are both accepted; Go's `flag` package
also accepts the single-dash long form (`-source`, `-target`). If a flag is
given more than once, the last occurrence wins.

`--agents` and `--skills` are mutually exclusive; passing both is a usage
error.

`--install NAME` and `--uninstall NAME` only make sense alongside
`--agents` or `--skills`; used with neither, either is a usage error, since
a bare name is ambiguous between an agent and a skill and an irreversible
operation must not guess. `NAME` is the bare item name, not a filename: an
agent stored as `agents/scout.md` is named `scout`, so `--install scout.md`
does not match anything — the `.md` suffix is stripped before comparing, so
agents and skills are named the same way. Matching is exact and
case-sensitive: no prefixes, globs, or fuzzy matching. Both flags are
repeatable, and a repeated name counts once. A `NAME` that matches nothing
is a fatal error (exit 1) rather than a run that quietly does nothing,
since an exact name matching nothing is almost always a typo.

`-i` only combines with `-s`/`-t`. Combining it with `--dry-run`,
`--agents`, `--skills`, `--install`, or `--uninstall` is a usage error —
everything those flags do is chosen inside the UI instead.

Exit codes: `0` on success, `1` on a usage error or a fatal error.

### Uninstalling

`--uninstall` removes items from the target. It is target-driven: what the
source contains has no bearing on what can be removed, which is why passing
`-s` alongside it is a usage error, and why a single run can never both
install and uninstall.

Every name is checked against the target **before anything is deleted**. If
any one of them matches nothing, the run fails and removes nothing — a typo
in the third name never leaves the first two already gone.

Removing a skill deletes the entire `<target>/skills/<name>/` directory,
including files inside it this tool never installed. A skill is one unit,
the same way installing already treats it. Removing an agent deletes
`<target>/agents/<name>.md`. If that empties `<target>/agents/` or
`<target>/skills/`, the empty directory is left in place.

`--dry-run` applies: it prints what would be removed and writes nothing.

### Examples

```
ai-config-manager -s ./claude -t ~/.claude
ai-config-manager -s ./claude -t ~/.claude --dry-run
ai-config-manager -i -s ./claude -t ~/.claude
ai-config-manager -s ./claude -t ~/.claude --agents
ai-config-manager -s ./claude -t ~/.claude --skills --install research
ai-config-manager -t ~/.claude --skills --uninstall research
ai-config-manager -t ~/.claude --agents --uninstall stale --uninstall older --dry-run
```

## Interactive mode

Passing `-i` opens a terminal UI for choosing what to install and what to
remove, instead of installing everything the source ships. It needs a real
interactive terminal: if stdin is redirected or piped, it exits `1` with an
error rather than hanging waiting for input.

The UI renders inline, in the same terminal directly below where the
command was run — it does not take over the screen or use the alternate
screen buffer.

There are four tabs: **Agents**, **Skills**, **Uninstall Agents**, and
**Uninstall Skills**.

The first two list what the **source** ships, grouped under three headings:

- **to update** — present in the target but different from the source.
- **to install** — not present in the target yet.
- **up to date** — byte-identical to the target already. These are shown
  but cannot be marked, since installing an identical item would do
  nothing.

The last two list what the **target** already holds, grouped under two
headings:

- **also in source** — the source ships an item of the same kind and name.
- **only in target** — stale items the source no longer ships, or ones put
  there by hand. The old two-tab UI could not see these at all.

Only items matching the tool's shape rules are listed: agents are `.md`
files directly in `<target>/agents`, skills are directories containing a
`SKILL.md` directly in `<target>/skills`. A missing or empty target
directory is not an error — the uninstall tab is simply empty.

Marking is multi-select and shared across all four tabs: everything marked
in any of them is applied together in one go when you confirm. Each tab's
label shows how many of its items are marked, and that count is signed —
`+` on an install tab means "will be written", `-` on an uninstall tab
means "will be erased".

The list scrolls when it's longer than the terminal, with `↑ N more` /
`↓ N more` markers when there's more above or below the visible window.

A search bar filters the current tab by a case-insensitive substring match
on the item name, and behaves identically in all four tabs: each tab
remembers its own search independently, and items filtered out of view stay
marked. This is a browse aid, and not the same thing as `--install` above —
`--install` is an exact, case-sensitive match on one name that skips the UI
entirely, while the search bar just narrows what's visible while you pick.

Marking the same item for install and for deletion at once is refused. A
conflict is the same kind plus the same name, so an agent named `research`
and a skill named `research` never conflict. The row shows `[!]` and
`(conflict)` in both tabs the instant it happens, and `enter` refuses to
proceed, listing every conflicting item instead. There is no override —
installing then deleting wastes the write, deleting then installing makes
the delete meaningless — so the only correct response is to unmark one
side.

Pressing enter opens a confirmation box listing everything currently
marked, in two sections: the deletions first, marked as destructive, then
the installs. Nothing is written or deleted until you confirm; on confirm,
the deletions run first, then the installs. After they finish and the UI
exits, a report is printed — see
[Interactive report](#interactive-report) below.

### Keys

| Key | Action |
| --- | --- |
| `j` / `k` (also ↓ / ↑) | Move the cursor down / up |
| `h` / `l` (also ← / →) | Previous / next tab |
| `space` | Mark or unmark the item under the cursor |
| `enter` | Apply the current marks (opens a confirmation first) |
| `/` | Open the search bar |
| `esc` | Close the search bar if open, else close the confirmation or conflict box if open, else quit |
| `ctrl+c` | Quit at any time |

With four tabs, `h` and `l` are genuinely opposite directions rather than
the same toggle; both wrap around at either end.

While the search bar is open, `j`, `k`, `h`, `l`, and `space` type into the
query instead of navigating — use the arrow keys to move the cursor while
searching. `left`/`right`, `backspace`, `delete`, `home`/`end`, and the
usual word-motions edit the query text. `enter` commits the search and
closes the bar without applying anything, so a stray enter while typing can
never install or remove anything. `esc` closes the search bar and clears
the query.

### Interactive report

The interactive report differs from the [normal one](#output). Each marked
item is reported as one of four statuses:

- **`installed`** — it was not present in the target before.
- **`updated`** — it was present in the target and different.
- **`removed`** — it was deleted from the target.
- **`failed`** — it could not be written or deleted, with the reason given.

Failed items are listed first, then removed, then installed, then updated,
alphabetically within each group. It ends with a summary line:
`N installed, M updated, K removed, L failed`.

Unlike a non-interactive run, which stops at the first error, an
interactive run does not stop at the first failure — every marked item is
attempted and accounted for in the report, and a deletion that fails is
reported `failed` with its reason, like any other item. The command exits
`1` if any item failed.

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

### Removal output

A removal run prints a report of the same shape:

```
$ ai-config-manager -t ~/.claude --skills --uninstall research --dry-run
DRY RUN — nothing removed

AGENTS
------
(none)

SKILLS
------
removed    research

1 removed
```

Same headings, same rule, same `(none)`, same status column. Every item in
it has the same status, `removed`, so there is no grouping within a
section — only a sort by name.

## What it will and will not touch

This is the part to trust before pointing it at a real config directory.

- The tool only ever installs into `<target>/agents` and `<target>/skills`,
  which it creates if missing. Within those, it writes only the items that
  come from `<source>/agents` and `<source>/skills`, one item at a time.
- **The tool reads and lists everything in `<target>/agents` and
  `<target>/skills`, and deletes only items you explicitly named with
  `--uninstall` or marked in the UI.** It still never reads, writes, or
  removes anything elsewhere in the target — Claude Code's own
  `settings.json`, `projects/`, `history.jsonl` and the rest stay
  untouched.
- Removal is by name and by name only. There is no prune mode, no "remove
  everything the source no longer ships", and nothing is ever deleted as a
  side effect of an install.
- Removing a skill deletes its whole directory, including files this tool
  never installed. `--dry-run`, the two-section confirmation, and the
  destructive marking in the UI are the safeguards.
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
  Re-running after fixing the cause is safe and will finish the job. A
  removal run likewise stops at the first failure and exits `1`; items
  already removed stay removed.

## Errors

The tool exits `1` with a fatal error in these cases:

- The source directory does not exist or is not a directory.
- The source contains neither an `agents/` nor a `skills/` subdirectory.
  (Having only one of the two is fine — that section is simply empty.)
- The source and target are the same directory, or one is nested inside
  the other, which would make the tool copy into its own input.
- A `--install` or `--uninstall` name matched nothing (in the source, or in
  the target, respectively). For `--uninstall`, this is checked for every
  name before anything is deleted, so nothing is removed.
