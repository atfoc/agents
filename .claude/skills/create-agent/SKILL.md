---
name: create-agent
description: Author a new subagent for this repo end to end — write the Claude Code version under claude/agents/, then port it to Cursor under cursor/agents/. Use when the user wants a brand-new subagent added to this repo, for both platforms, in one pass.
argument-hint: [what the agent should do]
disable-model-invocation: true
---

# Create a subagent for this repo — Claude Code, then Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or a description earlier in the conversation. If there is no subject, ask what the agent should do and stop.

## Step 1 — Author the Claude Code subagent

Read `${CLAUDE_PROJECT_DIR}/claude/skills/create-claude-agent/SKILL.md`, and its `reference.md` if it points you there. Follow those instructions yourself to write the new agent at `${CLAUDE_PROJECT_DIR}/claude/agents/<name>.md`, built from the subject. Do not stop to show it to the user yet — go straight to Step 2.

## Step 2 — Port it to Cursor

Read `${CLAUDE_PROJECT_DIR}/claude/skills/create-cursor-agent/SKILL.md` and follow it yourself to write the ported agent to `${CLAUDE_PROJECT_DIR}/cursor/agents/<name>.md`, translating the frontmatter to Cursor's documented fields and keeping the body's job and boundaries equivalent to the agent you just wrote in Step 1.

## Step 3 — Report

Show the user both files: `claude/agents/<name>.md` and `cursor/agents/<name>.md`.
