---
name: create-claude-agent
description: Create or revise a Claude Code subagent — the agent .md file, its frontmatter, and the system prompt that keeps it inside its job. Use when the user wants a new Claude Code subagent written, an existing one fixed, or an explanation of the Claude Code subagent format. Not for Cursor subagents — use create-cursor-agent for those.
disable-model-invocation: true
---

# Create a Claude Code subagent

A subagent is a single Markdown file: YAML frontmatter that configures it, and a body that is its system prompt. It runs in its own context window, sees none of the main conversation, and returns one final message to whoever spawned it.

The subject is whatever the user gave you: the skill argument, a description in the conversation, or an existing agent they want changed. If there is no subject, ask what the agent should do and stop.

## Step 1 — Settle what the agent is

A subagent earns its own file when its work is worth a separate context window and only the conclusion needs to come back. Searching twenty files, implementing one contained task, thinking hard about one decision — those qualify. A two-file lookup does not.

Settle these before writing anything:

- **The one job**, in a sentence starting "You are a …".
- **When the main agent should delegate to it**, and — just as important — **when it should not**.
- **What it may not do.** Read-only? Never touch tests? Never spawn other agents?
- **What its final message must contain**, since that is the only thing the caller ever sees.
- **The name.** Lowercase letters and hyphens, unique, no `:` (reserved for plugin-scoped identifiers).

## Step 2 — Put it in the right place

| Location                       | Scope                     |
| :----------------------------- | :------------------------ |
| Managed (enterprise) settings  | Organization-wide         |
| `--agents` CLI flag            | Current session           |
| `.claude/agents/`              | Current project           |
| `~/.claude/agents/`            | All your projects         |
| A plugin's `agents/` directory | Wherever the plugin is on |

When two definitions share a `name`, the one higher in that table wins. Directories are scanned recursively and names must be unique within one tree; with nested project directories defining the same name, the definition closest to the working directory is used.

## Step 3 — Write the frontmatter

`name` and `description` are the only required fields. Everything else is an override:

```yaml
---
name: scout
description: Read-only explorer. Use it for codebase questions and docs summaries. Not for making changes, decisions, or plans.
model: haiku
tools: Read, Grep, Glob, Bash
---
```

**`description` is routing, not documentation.** It is what the main agent reads when deciding where to send work, so write it as a rule: what this agent handles, and what it does not. An agent whose description omits the negative half gets over-delegated to.

Pick the model deliberately — a cheap model for mechanical exploration, a strong one with high `effort` for reasoning. The resolution order is: `CLAUDE_CODE_SUBAGENT_MODEL` env var, then `model` passed on the Agent tool call, then this file's frontmatter, then the main conversation's model. **A model or effort override on the spawn call outranks this file**, so a skill that depends on a pinned model must say "never pass a model or effort override".

The full field table — `tools`, `disallowedTools`, `effort`, `permissionMode`, `maxTurns`, `skills`, `mcpServers`, `hooks`, `memory`, `background`, `isolation`, `color`, `initialPrompt` — is in [reference.md](reference.md). Read it before using any field not shown above.

## Step 4 — Constrain what it can reach

Tools resolve in this order:

1. `disallowedTools` is applied first, removing tools from the pool.
2. `tools`, if set, is an allowlist over what remains.
3. If `tools` is omitted, the agent inherits every tool available to subagents, minus `disallowedTools`.

```yaml
tools: Read, Grep, Glob, Bash            # allowlist
disallowedTools: Write, Edit             # deny list
disallowedTools: mcp__*                  # drop all MCP tools
tools: Agent(worker, scout), Read, Bash  # restrict spawnable subagent types
```

Two things to plan around:

- **A subagent may spawn subagents.** `Agent` is only removed at the nesting depth limit, so an agent that should do its own work must be restricted through `tools` or told so in the body.
- **Interactive and orchestration tools are stripped from every subagent** regardless of frontmatter: `AskUserQuestion`, `EnterPlanMode`, `ExitPlanMode` (unless `permissionMode: plan`), `EndConversation`, `ScheduleWakeup`, `TaskOutput`, `Workflow`, `WaitForMcpServers`. Background subagents are restricted further, to file, shell, search, web, and MCP tools plus a few session tools. **A subagent cannot ask the user anything** — say in the body what it should do when it is stuck, because stopping to ask is not available to it.

## Step 5 — Write the body

The body replaces the main agent's system prompt entirely. The agent sees only this text and the delegation message it is handed, so it must be self-contained.

What earns its place:

- **The one job, in the first line.** `You are a worker. You are given exactly one task.`
- **Hard boundaries as checkable rules** — what it may not touch, what it may not run, when to stop. `Never run commands against the root of the filesystem (for example `find /`).`
- **The final-message contract.** Say explicitly that the caller sees only the final message, and list what it must carry: paths, line numbers, commands run and their real results, decisions made, blockers.
- **Report failures as failures.** Without that line, agents round their results up.

Keep it short. A page of constraints an agent can check itself against beats three pages of prose. This shape works:

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

## Step 6 — Check it before handing it over

- The description says both when to delegate and when not to.
- The body opens with the one job and closes with the final-message contract.
- Read-only agents are enforced through `tools`/`disallowedTools`, not only asked for in prose.
- The agent knows what to do when it is blocked, since it cannot ask.
- Field names are camelCase where multi-word: `disallowedTools`, `permissionMode`, `maxTurns`, `initialPrompt`, `mcpServers`. They are case-sensitive.

Show the user the file you wrote and the path it went to, then stop. Do not spawn the new agent to test it unless the user asks.

## Additional resources

- Full frontmatter reference, model strings, and preloaded skills: [reference.md](reference.md)
- Building this same agent for Cursor instead? Use the `create-cursor-agent` skill — the two formats share little beyond the concept.
