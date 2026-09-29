# agents

A personal collection of **skills** and **subagents** for AI coding agents, kept in one place and
shipped in two flavors: [Claude Code](https://claude.com/claude-code) and [Cursor](https://cursor.com).
It also includes **ai-config-manager**, a small CLI that installs them into your config directory.

## Layout

```
claude/            Skills and agents in Claude Code format
  agents/          Subagent definitions (one .md file each)
  skills/          Skills (one directory each, with a SKILL.md)
cursor/            The same skills and agents, ported to Cursor format
ai-config-manager/ Go CLI that installs the above into ~/.claude or ~/.cursor
.claude/skills/    Skills for working on this repo itself
```

`claude/` is the source of truth. Every skill and agent there has a Cursor port under `cursor/`
with the same name.

## ai-config-manager

Claude Code and Cursor pick up skills and agents from `~/.claude` and `~/.cursor`.
`ai-config-manager` copies them there from this repo — and removes ones you no longer want —
so you don't have to do it by hand.

It copies `<source>/agents/*.md` and `<source>/skills/*/` into the same paths under `<target>`.
It never touches anything else in the target (settings, history, projects), skips items that
are already identical, and only deletes what you name explicitly.

```sh
cd ai-config-manager
go build -o ai-config-manager .

# Install everything for Claude Code
./ai-config-manager -s ../claude -t ~/.claude

# See what would change without writing anything
./ai-config-manager -s ../claude -t ~/.claude --dry-run

# Pick what to install or remove in a terminal UI
./ai-config-manager -i -s ../claude -t ~/.claude

# Same for Cursor
./ai-config-manager -s ../cursor -t ~/.cursor
```

Requires Go 1.25+. See [ai-config-manager/README.md](ai-config-manager/README.md) for every
flag, the interactive UI, and exactly what it will and will not touch.

## Skills

A skill is a set of instructions (and sometimes bundled scripts) the agent loads when a task
matches it. You can also invoke one directly, e.g. `/define-feature` in Claude Code.

### From idea to shipped code

These chain together into a pipeline: define a feature, spec its implementation, break it into
tasks, and implement them.

| Skill | What it does |
| --- | --- |
| `define-feature` | Works out *what* a feature is (never *how*) by asking you one question at a time, and writes a feature definition document. |
| `build-doc-by-questions` | Builds any document — spec, plan, etc. — with you by asking one question at a time, each with a recommended answer. |
| `define-implementation-auto` | Turns a description of what to build into an implementation spec — modules, contracts, rules — without asking anything, settling open questions from the codebase's existing patterns. |
| `break-down-work` | Cuts a spec or large piece of work into vertical slices: a contracts slice first, then implementation slices each paired with a review slice. |
| `break-down-and-create-tasks` | Runs `break-down-work`, then creates the resulting tasks wherever you keep them (local folder, Linear, Asana). |
| `implement-tasks` | Implements every task marked `for-agent`: runs startable tasks in parallel subagents, each in its own git worktree, lands reviewed work and turns failed reviews into fix tasks until nothing is left. |
| `prototype` | Settles a question words can't by building it on its own branch — alone, or as variants you compare — and writes up the answer. |

### Task trackers

Used by the pipeline above to read and write tasks, and usable on their own.

| Skill | What it does |
| --- | --- |
| `manage-local-tasks` | Tasks as markdown files in a local folder: create, list, block, tag, complete, read. |
| `manage-linear-tasks` | Tasks as Linear issues and sub-issues, plus attached documents and file uploads. Needs `LINEAR_API_KEY`. |
| `manage-asana-tasks` | Tasks as Asana tasks and subtasks, plus attached markdown documents. Needs `ASANA_ACCESS_TOKEN`. |

### Other

| Skill | What it does |
| --- | --- |
| `generate-image` | Generates images from a text description via OpenRouter, optionally from reference images or with a transparent background. Needs `OPENROUTER_API_KEY`. |
| `excalidraw-live` | Creates, edits and reads `.excalidraw` drawings through a real Excalidraw engine in Chrome — headless, or in a browser tab you edit alongside the agent. |

## Agents

Subagents the main agent delegates to, each tuned to one kind of job.

| Agent | Role |
| --- | --- |
| `scout` | Read-only explorer. Answers one focused question with exact paths, lines and facts. Fast and cheap. |
| `thinker` | Hard reasoning. Produces a plan, settles a trade-off, or untangles many facts into one answer. Doesn't code. |
| `worker` | Implements one well-specified task — the code and its unit tests — and nothing beyond it. |

## Working on this repo

`.claude/skills/` holds skills for maintaining this repo. They are not installed anywhere.

| Skill | What it does |
| --- | --- |
| `create-skill` | Writes a new skill under `claude/skills/`, then ports it to `cursor/skills/`. |
| `create-agent` | Writes a new subagent under `claude/agents/`, then ports it to `cursor/agents/`. |
| `port-claude-skill` | Ports an existing Claude Code skill to Cursor (or refreshes the port). |
