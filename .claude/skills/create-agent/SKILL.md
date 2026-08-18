---
name: create-agent
description: Author a new subagent for this repo end to end — write the Claude Code version under claude/agents/, then port it to Cursor under cursor/agents/. Use when the user wants a brand-new subagent added to this repo, for both platforms, in one pass.
argument-hint: [what the agent should do]
disable-model-invocation: true
---

# Create a subagent for this repo — Claude Code, then Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or a description earlier in the conversation. If there is no subject, ask what the agent should do and stop.

## Constraint — do not use other agents or skills as reference

The only files you may read for guidance are the ones this skill names by path, plus any file those files point you to. Do not open other agents under `claude/agents/`, `cursor/agents/`, or other skills, to copy their structure, wording, or frontmatter, and do not treat an existing agent as a template. Write the new agent from the subject and the authoring instructions alone. The one exception is an agent or skill the user names explicitly as a reference — read that one.

## Step 1 — Author the Claude Code subagent

Read `${CLAUDE_PROJECT_DIR}/claude/skills/create-claude-agent/SKILL.md`, and its `reference.md` if it points you there. Follow those instructions yourself to write the new agent at `${CLAUDE_PROJECT_DIR}/claude/agents/<name>.md`, built from the subject. Do not stop to show it to the user yet — go straight to Step 2.

## Step 2 — Port it to Cursor in a subagent

Do not port it yourself. Spawn one subagent with the Agent tool, passing no `subagent_type`, and wait for it.

Its prompt is the two paths and the destination, nothing else:

```
Port the subagent at ${CLAUDE_PROJECT_DIR}/claude/agents/<name>.md to Cursor.
The Cursor subagent format is documented at ${CLAUDE_PROJECT_DIR}/claude/skills/create-cursor-agent/SKILL.md — read it and follow it.
Write the ported agent to ${CLAUDE_PROJECT_DIR}/cursor/agents/<name>.md.
```

Substitute `<name>` and expand `${CLAUDE_PROJECT_DIR}` to the real path before sending it. Do not paste the agent's content into the prompt, do not summarise what the agent does, and do not add porting rules of your own — the subagent reads both files itself.

## Step 3 — Report

Show the user both files: `claude/agents/<name>.md` and `cursor/agents/<name>.md`.
