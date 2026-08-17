---
name: create-skill
description: Author a new skill for this repo end to end — write the Claude Code version under claude/skills/, then port it to Cursor under cursor/skills/. Use when the user wants a brand-new skill added to this repo, for both platforms, in one pass.
argument-hint: [what the skill should do]
disable-model-invocation: true
---

# Create a skill for this repo — Claude Code, then Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or a description earlier in the conversation. If there is no subject, ask what the skill should do and stop.

## Step 1 — Author the Claude Code skill

Read `${CLAUDE_PROJECT_DIR}/claude/skills/create-claude-skill/SKILL.md`, and its `reference.md` if it points you there. Follow those instructions yourself to write the new skill at `${CLAUDE_PROJECT_DIR}/claude/skills/<name>/SKILL.md` (plus any sibling files it calls for), built from the subject. Do not stop to show it to the user yet — go straight to Step 2.

## Step 2 — Port it to Cursor

Read `${CLAUDE_PROJECT_DIR}/claude/skills/create-cursor-skill/SKILL.md` and follow it yourself to write the ported skill to `${CLAUDE_PROJECT_DIR}/cursor/skills/<name>/SKILL.md`, translating the frontmatter to Cursor's five documented fields and keeping the body's trigger and behavior equivalent to the skill you just wrote in Step 1.

## Step 3 — Report

Show the user both files: `claude/skills/<name>/SKILL.md` and `cursor/skills/<name>/SKILL.md`.
