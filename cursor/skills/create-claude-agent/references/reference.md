# Claude Code subagent format — full reference

Read this when `SKILL.md` points you here: every optional frontmatter field, how models resolve,
and the failure modes worth knowing before you use one. This is Claude Code's format, not
Cursor's — none of it belongs in a Cursor agent file.

## Required fields

| Field         | Rules |
| :------------ | :---- |
| `name`        | Unique identifier, lowercase letters and hyphens. Cannot contain `:` (reserved for plugin-scoped identifiers). |
| `description` | When Claude should delegate to this subagent. This is the text the main agent reads when deciding, so write it as a routing rule: what the agent is for, and what it is *not* for. |

## Optional fields

| Field             | Type                 | Default     | What it does |
| :---------------- | :------------------- | :---------- | :----------- |
| `tools`           | comma-separated list | inherit all | Allowlist of tools. Accepts `mcp__<server>` and `mcp__<server>__*` patterns, and `Agent(type1, type2)` to restrict which subagent types this one may spawn. |
| `disallowedTools` | comma-separated list | none        | Tools to deny. Applied before `tools`. |
| `model`           | string               | `inherit`   | `sonnet`, `opus`, `haiku`, `fable`, a full model ID (e.g. `claude-opus-5`), or `inherit`. |
| `effort`          | string               | inherit     | `low`, `medium`, `high`, `xhigh`, `max`. Overrides the session effort level. |
| `permissionMode`  | string               | inherit     | `default`, `acceptEdits`, `auto`, `dontAsk`, `bypassPermissions`, or `plan`. Ignored for plugin subagents. |
| `maxTurns`        | number               | unlimited   | Hard stop after N agentic turns. |
| `skills`          | list of strings      | none        | Skills preloaded into the subagent at startup — the **full** skill content is injected, not just the description. |
| `mcpServers`      | list                 | none        | MCP servers available to this subagent (names, or inline definitions). Ignored for plugin subagents. |
| `hooks`           | object               | none        | Lifecycle hooks scoped to this subagent. Ignored for plugin subagents. |
| `memory`          | string               | none        | Persistent memory scope: `user`, `project`, or `local`. |
| `background`      | boolean              | `false`     | Keep this subagent in the background even when the main agent asks for the foreground. |
| `isolation`       | string               | none        | `worktree` runs it in a temporary git worktree with an isolated copy of the repo. |
| `color`           | string               | default     | `red`, `blue`, `green`, `yellow`, `purple`, `orange`, `pink`, `cyan`. Display only. |
| `initialPrompt`   | string               | none        | Auto-submitted as the first user turn when this agent runs as the main session (`--agent`). |

Field names are case-sensitive and camelCase where multi-word. None of this table is available to
you when writing this skill's own `SKILL.md` — Cursor caps that file at five fields. It only
describes the *agent file* the skill produces.

## How the model is chosen

In order, first match wins:

1. `CLAUDE_CODE_SUBAGENT_MODEL` environment variable
2. `model` passed on the Agent tool call
3. `model` in this file's frontmatter
4. The main conversation's model

Aliases (`sonnet`, `opus`, `haiku`, `fable`) are the portable choice; full IDs pin an exact model
and can carry variant brackets, as in `claude-sonnet-5[1m]` for the 1M-context variant.

Because a spawn-call override outranks the file, any skill built around a specific agent should
tell the caller not to pass `model` or `effort` on the spawn call.

## Preloaded skills

`skills` injects the **entire body** of each named skill into the subagent at startup, not just
its description. That is the point — the agent gets the procedure without having to decide to load
it — but it is charged in full context every time the agent starts. Preload one procedure the
agent always needs; never preload a library of them.

A skill marked `disable-model-invocation: true` is excluded from subagent preloading.

## Isolation

`isolation: worktree` gives the agent its own git worktree — an isolated copy of the repo — and
the worktree is cleaned up automatically if nothing changed. It costs setup time and disk per
agent, so use it only when several agents mutate files concurrently and would otherwise collide.

## Things that trip people up

- **`description` is routing, not documentation.** If it doesn't say when *not* to use the agent,
  the main agent will over-delegate to it.
- **A subagent may spawn subagents** unless you stop it — `Agent` is only removed at the nesting
  depth limit. Restrict `tools` or say so in the body.
- **A subagent cannot ask the user anything.** `AskUserQuestion` is stripped from every subagent,
  so the body has to say what to do when the task is ambiguous or blocked.
- **The caller sees only the final message.** Anything the agent discovered but did not write
  there is lost when its context is discarded.
- **Preloaded `skills` cost context at startup**, because the whole skill body is injected.
- **`permissionMode`, `mcpServers` and `hooks` are ignored for plugin subagents.**

## Cross-tool portability

Cursor reads `.claude/agents/` as a compatibility path, but supports only `name`, `description`,
`model`, `readonly` and `is_background` — and its model strings differ (`claude-opus-5[effort=high]`
rather than a separate `effort` field). An agent meant to work in both places needs a separate
file per tool; keep the body shared and let the frontmatter differ. The `create-cursor-agent`
skill documents that side of the pair in full.
