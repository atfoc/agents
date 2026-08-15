# Claude Code — subagent format

A subagent is a single Markdown file: YAML frontmatter that configures it, and a body that
is its system prompt. This document describes what Claude Code accepts in that file.

The subagents in this repo live in `claude/agents/` (`scout.md`, `thinker.md`, `worker.md`)
and are installed into a project's `.claude/agents/`.

## Where the files live

Claude Code discovers subagents from several places. When two definitions share a `name`,
the one higher in this table wins.

| Location                       | Scope                        |
| :----------------------------- | :--------------------------- |
| Managed (enterprise) settings  | Organization-wide            |
| `--agents` CLI flag            | Current session              |
| `.claude/agents/`              | Current project              |
| `~/.claude/agents/`            | All your projects            |
| A plugin's `agents/` directory | Wherever the plugin is on    |

Directories are scanned recursively, and names must be unique within one directory tree.
With nested project directories defining the same name, the definition closest to the
working directory is used.

## Minimal file

```markdown
---
name: scout
description: Read-only explorer. Use it for codebase questions and docs summaries. Not for making changes.
---

You are a scout. You explore and you report. You never change anything.

- Read-only: do not create, edit or delete any file.
- Be specific: exact file paths, line numbers, names, short quoted snippets.

Your final message is the summary, and it is the only thing the caller sees.
```

`name` and `description` are the only required fields. Everything else is an override.

## Required fields

| Field         | Rules                                                                                  |
| :------------ | :------------------------------------------------------------------------------------- |
| `name`        | Unique identifier, lowercase letters and hyphens. Cannot contain `:` (reserved for plugin-scoped identifiers). |
| `description` | When Claude should delegate to this subagent. This is the text the main agent reads when deciding, so write it as a routing rule: what the agent is for, and what it is *not* for. |

## Optional fields

| Field              | Type                 | Default     | What it does |
| :----------------- | :------------------- | :---------- | :----------- |
| `tools`            | comma-separated list | inherit all | Allowlist of tools. Accepts `mcp__<server>` and `mcp__<server>__*` patterns, and `Agent(type1, type2)` to restrict which subagent types this one may spawn. |
| `disallowedTools`  | comma-separated list | none        | Tools to deny. Applied before `tools`. |
| `model`            | string               | `inherit`   | `sonnet`, `opus`, `haiku`, `fable`, a full model ID (e.g. `claude-opus-5`), or `inherit`. |
| `effort`           | string               | inherit     | `low`, `medium`, `high`, `xhigh`, `max`. Overrides the session effort level. |
| `permissionMode`   | string               | inherit     | `default`, `acceptEdits`, `auto`, `dontAsk`, `bypassPermissions`, or `plan`. Ignored for plugin subagents. |
| `maxTurns`         | number               | unlimited   | Hard stop after N agentic turns. |
| `skills`           | list of strings      | none        | Skills preloaded into the subagent at startup — the **full** skill content is injected, not just the description. |
| `mcpServers`       | list                 | none        | MCP servers available to this subagent (names, or inline definitions). Ignored for plugin subagents. |
| `hooks`            | object               | none        | Lifecycle hooks scoped to this subagent. Ignored for plugin subagents. |
| `memory`           | string               | none        | Persistent memory scope: `user`, `project`, or `local`. |
| `background`       | boolean              | `false`     | Keep this subagent in the background even when the main agent asks for the foreground. |
| `isolation`        | string               | none        | `worktree` runs it in a temporary git worktree with an isolated copy of the repo. |
| `color`            | string               | default     | `red`, `blue`, `green`, `yellow`, `purple`, `orange`, `pink`, `cyan`. Display only. |
| `initialPrompt`    | string               | none        | Auto-submitted as the first user turn when this agent runs as the main session (`--agent`). |

## How tools resolve

1. `disallowedTools` is applied first, removing tools from the pool.
2. `tools`, if set, is an allowlist over what remains.
3. If `tools` is omitted, the subagent inherits every tool available to subagents, minus
   anything in `disallowedTools`.

Some tools are stripped from every subagent regardless of frontmatter — the interactive and
orchestration ones: `AskUserQuestion`, `EnterPlanMode`, `ExitPlanMode` (unless
`permissionMode: plan`), `EndConversation`, `ScheduleWakeup`, `TaskOutput`, `Workflow`,
`WaitForMcpServers`, and `Agent` at the nesting depth limit.

Background subagents are restricted further, to file, shell, search, web, and MCP tools plus
a few session tools.

Examples:

```yaml
tools: Read, Grep, Glob, Bash          # allowlist
disallowedTools: Write, Edit           # deny list
disallowedTools: mcp__*                # drop all MCP tools
tools: Agent(worker, scout), Read, Bash  # restrict spawnable subagent types
```

## How the model is chosen

In order, first match wins:

1. `CLAUDE_CODE_SUBAGENT_MODEL` environment variable
2. `model` passed on the Agent tool call
3. `model` in this file's frontmatter
4. The main conversation's model

Aliases (`sonnet`, `opus`, `haiku`, `fable`) are the portable choice; full IDs pin an exact
model and can carry variant brackets, as in `claude-sonnet-5[1m]` for the 1M-context variant.

## Writing the body

The body becomes the subagent's system prompt. It replaces the main agent's prompt — the
subagent does not see your conversation, only the delegation message it is given.

What earns its place there:

- The one job, stated in the first line. `You are a worker. You are given exactly one task.`
- Hard boundaries, phrased as rules the agent can check itself against: what it may not touch,
  what it may not run, when to stop.
- What the final message must contain. The caller sees **only** the final message, so say so
  explicitly and list what it has to carry.

This repo's three agents follow that shape — a one-line identity, a short bullet list of
constraints, then a closing paragraph defining the final message.

## Full example

```markdown
---
name: worker
description: Implements one focused, well-specified task — the code and its unit tests. Give it a single self-contained job stating the goal, the files it owns, and what done looks like. Not for exploration, planning, or open-ended work.
model: claude-sonnet-5[1m]
effort: xhigh
---

You are a worker. You are given exactly one task. You implement it fully.

- Do the whole task, and nothing beyond it.
- Only touch the files your task says you own.
- Match the surrounding code: its style, naming, structure, error handling and comment density.
- Never run commands against the root of the filesystem (for example `find /`).

Your final message is a short report, and it is the only thing the caller sees: files changed,
decisions you made, commands you ran and their real results, and anything that blocks the rest
of the work. Report failures as failures.
```

## Things that trip people up

- **`description` is routing, not documentation.** If it doesn't say when *not* to use the
  agent, the main agent will over-delegate to it.
- **A subagent may spawn subagents** unless you stop it — `Agent` is only removed at the
  nesting depth limit. If an agent should do its own work, restrict `tools` or say so in the body.
- **Preloaded `skills` cost context at startup**, because the whole skill body is injected.
- **Field names are case-sensitive** and use camelCase where multi-word (`disallowedTools`,
  `permissionMode`, `maxTurns`, `initialPrompt`, `mcpServers`).
