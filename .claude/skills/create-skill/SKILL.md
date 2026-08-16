---
name: create-skill
description: Author a new skill for this repo end to end — write the Claude Code version under claude/skills/, then delegate to a worker subagent to port it to Cursor under cursor/skills/. Use when the user wants a brand-new skill added to this repo, for both platforms, in one pass.
argument-hint: [what the skill should do]
disable-model-invocation: true
---

# Create a skill for this repo — Claude Code, then Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or a description earlier in the conversation. If there is no subject, ask what the skill should do and stop.

## Step 1 — Author the Claude Code skill

Read `${CLAUDE_PROJECT_DIR}/claude/skills/create-claude-skill/SKILL.md`, and its `reference.md` if it points you there. Follow those instructions yourself to write the new skill at `${CLAUDE_PROJECT_DIR}/claude/skills/<name>/SKILL.md` (plus any sibling files it calls for), built from the subject. Do not stop to show it to the user yet — go straight to Step 2.

## Step 2 — Port it to Cursor with a worker subagent

Launch a subagent of type `worker` to port the skill you just wrote. Its prompt must contain only:

- The path(s) of the skill file(s) you wrote in Step 1, with an instruction to Read them.
- An instruction to read `${CLAUDE_PROJECT_DIR}/claude/skills/create-cursor-skill/SKILL.md` and follow it.
- The goal: write the ported skill to `${CLAUDE_PROJECT_DIR}/cursor/skills/<name>/SKILL.md`, translating the frontmatter to Cursor's five documented fields and keeping the body's trigger and behavior equivalent.

Do not explain the Claude Code skill format, the create-claude-skill steps, or your own reasoning from Step 1 to the subagent — it starts with zero context, and it should only ever see the new skill file and create-cursor-skill.

## Step 3 — Report

Once the subagent returns, show the user both files: `claude/skills/<name>/SKILL.md` and `cursor/skills/<name>/SKILL.md`.
