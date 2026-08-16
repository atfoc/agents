---
name: create-agent
description: Author a new subagent for this repo end to end — write the Claude Code version under claude/agents/, then delegate to a worker subagent to port it to Cursor under cursor/agents/. Use when the user wants a brand-new subagent added to this repo, for both platforms, in one pass.
argument-hint: [what the agent should do]
disable-model-invocation: true
---

# Create a subagent for this repo — Claude Code, then Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or a description earlier in the conversation. If there is no subject, ask what the agent should do and stop.

## Step 1 — Author the Claude Code subagent

Read `${CLAUDE_PROJECT_DIR}/claude/skills/create-claude-agent/SKILL.md`, and its `reference.md` if it points you there. Follow those instructions yourself to write the new agent at `${CLAUDE_PROJECT_DIR}/claude/agents/<name>.md`, built from the subject. Do not stop to show it to the user yet — go straight to Step 2.

## Step 2 — Port it to Cursor with a worker subagent

Launch a subagent of type `worker` to port the agent you just wrote. Its prompt must contain only:

- The path of the agent file you wrote in Step 1, with an instruction to Read it.
- An instruction to read `${CLAUDE_PROJECT_DIR}/claude/skills/create-cursor-agent/SKILL.md` and follow it.
- The goal: write the ported agent to `${CLAUDE_PROJECT_DIR}/cursor/agents/<name>.md`, translating the frontmatter to Cursor's documented fields and keeping the body's job and boundaries equivalent.

Do not explain the Claude Code subagent format, the create-claude-agent steps, or your own reasoning from Step 1 to the subagent — it starts with zero context, and it should only ever see the new agent file and create-cursor-agent.

## Step 3 — Report

Once the subagent returns, show the user both files: `claude/agents/<name>.md` and `cursor/agents/<name>.md`.
