---
name: create-cursor-agent
description: Create or revise a Cursor subagent — the agent .md file, its frontmatter, and the system prompt that keeps it inside its job. Use when the user wants a Cursor subagent written, an existing one fixed, or an explanation of the Cursor subagent format. Not for Claude Code subagents — use create-claude-agent for those.
disable-model-invocation: true
---

# Create a Cursor subagent

You are working inside Claude Code, but the file you are about to write is not for Claude Code — it is for Cursor, and Cursor's subagent format is its own thing. Do not reach for Claude Code conventions (`tools`, `disallowedTools`, `effort`, `permissionMode`, `hooks`, and the rest) here; none of them apply. What follows is the complete Cursor format.

A Cursor subagent is a single Markdown file: YAML frontmatter that configures it, and a body that is its system prompt. It runs with its own context window, works autonomously on the prompt it is given, and returns a final message with its results.

The subject is whatever the user gave you: the skill argument, a description in the conversation, or an existing agent they want changed. If there is no subject, ask what the agent should do and stop.

## Step 1 — Settle what the agent is

A subagent earns its own file when its work is worth a separate context window and only the conclusion needs to come back. Searching twenty files, implementing one contained task, thinking hard about one decision — those qualify. A two-file lookup does not.

Settle these before writing anything:

- **The one job**, in a sentence starting "You are a …".
- **When the main agent should delegate to it**, and — just as important — **when it should not**.
- **What it may not do.** Read-only? Never touch tests? Never spawn another subagent?
- **What its final message must contain**, since that is the only thing the caller ever sees.
- **The name.** Lowercase letters and hyphens.

## Step 2 — Put it in the right place

Project level:

- `.cursor/agents/`
- `.claude/agents/` (compatibility path)
- `.codex/agents/` (compatibility path)

User level:

- `~/.cursor/agents/`
- `~/.claude/agents/` (compatibility path)
- `~/.codex/agents/` (compatibility path)

Project subagents take precedence over user ones, and `.cursor/` takes precedence over the compatibility paths. Inside a plugin, agents are `.md`, `.mdc` or `.markdown` files in the plugin's `agents/` directory, discovered automatically.

## Step 3 — Write the frontmatter

Every field is optional, and this is the complete documented set:

| Field           | Type    | Default               | Values / meaning |
| :-------------- | :------ | :--------------------- | :--------------- |
| `name`          | string  | derived from filename  | Lowercase letters and hyphens. |
| `description`   | string  | —                       | What the agent is for. This drives automatic delegation, so write it as a routing rule: what it handles, and what it does not. |
| `model`         | string  | `inherit`               | `inherit`, or a model ID, optionally with parameters. |
| `readonly`      | boolean | `false`                 | `true` prevents the subagent from modifying anything. |
| `is_background` | boolean | `false`                 | `true` runs the subagent in the background without blocking the parent. |

Note the field name: `is_background`, with underscores, unlike the hyphenated names used elsewhere in Cursor configuration.

**`description` is routing, not documentation.** It is what the main agent reads when deciding where to send work, so write it as a rule: what this agent handles, and what it does not. An agent whose description omits the negative half gets over-delegated to.

**`model`** takes either `inherit`, which uses the parent agent's model, or a specific model ID such as `composer-2` or `gpt-5.6-sol`. Parameters go in brackets after the ID, comma-separated — the documented ones are `fast`, `effort` and `context`:

```yaml
model: claude-opus-5[effort=high,context=300k]
```

There is no separate effort field here; effort is a model parameter. Model IDs change as models ship, so check Cursor's model picker for what is currently valid before copying an ID into a new agent, and use `inherit` when you do not care which model runs it.

## Step 4 — Know what the format will not do for you

Cursor documents exactly those five fields, so anything else you may be used to writing in a Claude Code agent file has no effect:

- **No tool allowlist or deny list.** There is no `tools` or `allowed-tools` field. The only capability control is `readonly: true`, which is coarse: read-only or not. Everything finer — "never delete", "never run this command" — has to be a rule in the body, and is honoured by instruction rather than enforced.
- **No effort field.** Effort goes inside the bracket syntax (`[effort=high]`).
- **No per-agent hooks, MCP server lists, permission modes, turn limits, memory scopes, worktree isolation, or display colour.**

Cursor's docs do not say what happens to unrecognised frontmatter keys, so do not rely on extra fields being either honoured or harmlessly ignored. Keep agent files to the documented five and put the rest in the body, where it will actually be read.

**Nesting stops after one level.** A subagent launched by another subagent cannot launch further ones. Plan for it: an agent that assumes it can fan work out to helpers will simply have to do that work itself, and it must be told so in its body. Skills invoked from a subagent inherit the same limit.

## Step 5 — Write the body

The body is the subagent's system prompt. It does not see the conversation — only the prompt it is handed — so it must be self-contained.

What earns its place:

- **The one job, in the first line.** `You are a worker. You are given exactly one task.`
- **Hard boundaries as checkable rules** — what it may not touch, what it may not run, when to stop. `Never run commands against the root of the filesystem (for example `find /`).`
- **The no-nesting rule**, whenever the agent might otherwise try to delegate: "Do the work yourself. Never launch another subagent."
- **Every restriction the frontmatter cannot enforce**, since `readonly` is the only real control.
- **The final-message contract.** Say explicitly that the caller sees only the final message, and list what it must carry: paths, line numbers, commands run and their real results, decisions made, blockers.
- **Report failures as failures.** Without that line, agents round their results up.

Keep it short. A page of constraints an agent can check itself against beats three pages of prose. This shape works:

```markdown
---
name: worker
description: Implements one focused, well-specified task — the code and its unit tests. Give it a single self-contained job stating the goal, the files it owns, and what done looks like. Not for exploration, planning, or open-ended work.
model: claude-sonnet-5-thinking-xhigh
---

You are a worker. You are given exactly one task. You implement it fully.

- Do the whole task, and nothing beyond it.
- Only touch the files your task says you own.
- Match the surrounding code: its style, naming, structure, error handling and comment density.
- Never run commands against the root of the filesystem (for example `find /`).
- Do the work yourself. Never launch another subagent.

Your final message is a short report, and it is the only thing the caller sees: files changed,
decisions you made, commands you ran and their real results, and anything that blocks the rest
of the work. Report failures as failures.
```

## Step 6 — Check it before handing it over

- The description says both when to delegate and when not to.
- Only the five documented fields appear in the frontmatter, `is_background` spelled with underscores.
- The body opens with the one job and closes with the final-message contract.
- A read-only agent sets `readonly: true` *and* says so in the body.
- The no-nesting rule is present if the agent could otherwise try to delegate.

Show the user the file you wrote and the path it went to, then stop. Do not launch the new agent to test it unless the user asks.

Building this same agent for Claude Code instead? Use the `create-claude-agent` skill — Claude Code's format has roughly a dozen fields Cursor's does not.
