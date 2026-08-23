---
name: create-skill
description: Author a new skill for this repo end to end — write the Claude Code version under claude/skills/, then port it to Cursor under cursor/skills/. Use when the user wants a brand-new skill added to this repo, for both platforms, in one pass.
argument-hint: [what the skill should do]
disable-model-invocation: true
---

# Create a skill for this repo — Claude Code, then Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or a description earlier in the conversation. If there is no subject, ask what the skill should do and stop.

## Constraint — do not use other skills as reference

The only files you may read for guidance are the ones this skill names by path, plus any file those files point you to. Do not open other skills under `claude/skills/`, `cursor/skills/`, or anywhere else to copy their structure, wording, or frontmatter, and do not treat an existing skill as a template. Write the new skill from the subject and the authoring instructions alone. The one exception is a skill the user names explicitly as a reference — read that one.

## Step 1 — Author the Claude Code skill

Read `${CLAUDE_PROJECT_DIR}/claude/skills/create-claude-skill/SKILL.md`, and its `reference.md` if it points you there. Follow those instructions yourself to write the new skill at `${CLAUDE_PROJECT_DIR}/claude/skills/<name>/SKILL.md` (plus any sibling files it calls for), built from the subject. Do not stop to show it to the user yet — go straight to Step 2.

## Step 2 — Port it to Cursor in a subagent

Do not port it yourself. Spawn one subagent with the Agent tool, passing no `subagent_type`, and wait for it.

Its prompt is the two paths and the destination, nothing else:

```
Port the skill at ${CLAUDE_PROJECT_DIR}/claude/skills/<name>/ to Cursor.
The Cursor skill format is documented at ${CLAUDE_PROJECT_DIR}/claude/skills/create-cursor-skill/SKILL.md — read it and follow it.
Write the ported skill to ${CLAUDE_PROJECT_DIR}/cursor/skills/<name>/.
Do not reword skill. Content of skill should stay as is. Only change things that are not supported in cursor version.
```

Substitute `<name>` and expand `${CLAUDE_PROJECT_DIR}` to the real path before sending it. Do not paste the skill's content into the prompt, do not summarise what the skill does, and do not add porting rules of your own — the subagent reads both files itself.
When porting text of skill should stay as is in claude version. Subagent should not reword it. Only replace thing in body
if they are not supported in cursor

## Step 3 — Report

Show the user both files: `claude/skills/<name>/SKILL.md` and `cursor/skills/<name>/SKILL.md`.
