# Cursor — subagent format

A Cursor subagent is a single Markdown file: YAML frontmatter that configures it, and a body
that is its system prompt. Each subagent runs with its own context window, works autonomously
on the prompt it is given, and returns a final message with its results.

The subagents in this repo live in `cursor/agents/` (`scout.md`, `thinker.md`, `worker.md`) and
are installed into a project's `.cursor/agents/` by `cursor/install-cursor-config.sh`.

## Where the files live

Project level:

- `.cursor/agents/`
- `.claude/agents/` (compatibility path)
- `.codex/agents/` (compatibility path)

User level:

- `~/.cursor/agents/`
- `~/.claude/agents/` (compatibility path)
- `~/.codex/agents/` (compatibility path)

Project subagents take precedence over user ones, and `.cursor/` takes precedence over the
compatibility paths. Inside a plugin, agents are `.md`, `.mdc`, or `.markdown` files in the
plugin's `agents/` directory, discovered automatically (this repo ships them that way, with the
manifest in `cursor/.cursor-plugin/plugin.json`).

## Minimal file

```markdown
---
name: scout
description: Read-only explorer. Use it for codebase questions and docs summaries. Not for making changes.
model: composer-2.5
readonly: true
---

You are a scout. You explore and you report. You never change anything.

- Read-only: do not create, edit or delete any file.
- Be specific: exact file paths, line numbers, names, short quoted snippets.
- Do your own looking. Never launch another subagent.

Your final message is the summary, and it is the only thing the caller sees.
```

## Frontmatter reference

Every field is optional. This is the complete documented set.

| Field           | Type    | Default              | Values / meaning |
| :-------------- | :------ | :------------------- | :--------------- |
| `name`          | string  | derived from filename | Lowercase letters and hyphens. |
| `description`   | string  | —                    | What the agent is for. This is what drives automatic delegation, so write it as a routing rule: what it handles, and what it does not. |
| `model`         | string  | `inherit`            | `inherit`, or a model ID, optionally with parameters. |
| `readonly`      | boolean | `false`              | `true` prevents the subagent from modifying anything. |
| `is_background` | boolean | `false`              | `true` runs the subagent in the background without blocking the parent. |

Note the field name: `is_background`, with underscores, unlike the hyphenated names used
elsewhere in Cursor configuration.

## Model strings

`model` accepts either `inherit`, which uses the parent agent's model, or a specific model ID
such as `composer-2` or `gpt-5.6-sol`. Parameters go in brackets after the ID, comma-separated:

```yaml
model: claude-opus-5[effort=high,context=300k]
```

The documented parameters are `fast`, `effort`, and `context`.

The agents in this repo pin models directly — `composer-2.5` for the scout,
`claude-sonnet-5-thinking-xhigh` for the worker, `claude-opus-5-thinking-xhigh` for the thinker.
Model IDs change as models ship; check Cursor's model picker for what is currently valid before
copying an ID into a new agent, and use `inherit` when you don't care which model runs it.

## How subagents get invoked

Three ways:

- **Automatic delegation.** The main agent decides, based on task complexity and scope, the
  subagent descriptions available in the project, and the current context.
- **`/name` in your prompt**, to request a specific subagent.
- **Mentioning it naturally** in your prompt.

Several subagents can be launched at once and run in parallel.

## The nesting limit

A subagent launched by another subagent cannot launch further ones — nesting stops after one
level. Plan for it: a subagent that assumes it can fan work out to helpers will simply have to
do that work itself, and it should be told so.

That is exactly why each agent in this repo carries an explicit line in its body — "Do your own
looking. Never launch another subagent." for the scout, "Do the work yourself." for the worker,
"Reason it out yourself." for the thinker. Skills invoked from a subagent inherit the same
limit, so the same instruction belongs anywhere delegation is described.

## Writing the body

The body is the subagent's system prompt. It does not see your conversation — only the prompt
it is handed — so it must be self-contained.

What earns its place:

- The one job, stated in the first line. `You are a worker. You are given exactly one task.`
- Hard boundaries phrased as checkable rules: what it may not touch, what it may not run,
  when to stop.
- The no-nesting rule, whenever the agent might otherwise try to delegate.
- What the final message must contain, since the caller sees only that.

## Full example

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

## What this format does not give you

Cursor documents exactly the five fields above, so anything else you may be used to writing in
an agent file has no effect here:

- **No tool allowlist or deny list.** There is no `tools` or `allowed-tools` field. The only
  capability control is `readonly: true`, which is coarse: read-only or not. Everything finer —
  "never delete", "never run this command" — has to be a rule in the body, and is honoured by
  instruction rather than enforced.
- **No effort field.** Effort is a model parameter here, written inside the bracket syntax
  (`[effort=high]`), not as its own frontmatter key.
- **No per-agent hooks, MCP server lists, permission modes, turn limits, memory scopes,
  worktree isolation, or display colour.**

Cursor's docs don't say what happens to unrecognised frontmatter keys, so don't rely on extra
fields being either honoured or harmlessly ignored — keep agent files to the documented five,
and put the rest in the body where it will actually be read.
