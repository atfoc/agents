---
name: create-skill
description: Authors a new skill for this repo end to end, writing the Claude Code version under claude/skills/ and then porting it to Cursor under cursor/skills/. Use when you want to add a new skill to this repo.
argument-hint: [what the skill should do]
---

# Create a skill for this repo — Claude Code, then Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or a description earlier in the conversation. If there is no subject, ask what the skill should do and stop.

## Constraint — do not use other skills as reference

Do not open other skills under `claude/skills/`, `cursor/skills/`, or anywhere else to copy their structure, wording, or frontmatter, and do not treat an existing skill as a template. Write the new skill from the subject and Step 1 alone. The one exception is a skill the user names explicitly as a reference — read that one.

## Step 1 — Author the Claude Code skill

A skill is a folder at `${CLAUDE_PROJECT_DIR}/claude/skills/<name>/` holding a `SKILL.md` and, optionally, sibling files. If that folder already exists, tell the user and stop — this skill creates, it does not overwrite.

**Name.** Lowercase letters, digits and hyphens, short, naming the action: `generate-image`, not `image-helper`. The folder name and the `name` field are identical.

**Frontmatter.** A YAML block between `---` lines at the top of `SKILL.md`:

- `name` — required, as above.
- `description` — required. Always two parts, in this order: what the skill does, then a `Use when …` sentence naming the situation that should trigger it, in the words a user would type. One or two sentences in total. This holds whether or not the skill sets `disable-model-invocation`. The port to Cursor carries it over verbatim, so get it right here.
- `argument-hint` — when the skill takes input, a short bracketed hint such as `[file path]`.
- `disable-model-invocation: true` — only when the skill must run solely because the user asked for it by name, such as one with side effects the user should decide on. Leave it out otherwise so the model can invoke the skill.
- `allowed-tools` — only when the skill needs tools pre-approved to do its job.

Add no other fields unless the subject calls for one.

**Body.** Markdown after the frontmatter:

1. A `#` heading naming the task.
2. A first paragraph saying what the skill acts on — `$ARGUMENTS`, or what was named earlier in the conversation — and what to do when that is missing, usually: ask for it and stop.
3. The instructions, as numbered steps or short sections, written as direct commands. Say what to do, not why skills exist. State hard rules plainly and once.
4. A last line saying when the skill is finished and that it stops there.

**Sibling files.** Put executable helpers in `scripts/`, long reference material the body only sometimes needs in `references/`, and templates or static files in `assets/`. Reference them from the body as `${CLAUDE_SKILL_DIR}/scripts/<file>` and so on. Keep `SKILL.md` focused; move bulk out to sibling files rather than inlining it.

Write the skill, then go straight to Step 2 without showing it to the user.

## Step 2 — Port it to Cursor in a subagent

Do not port it yourself. Spawn one subagent with the Agent tool, run the following prompt in it, and wait for it to finish:

```
Port skill from Claude to Cursor: ${CLAUDE_PROJECT_DIR}/claude/skills/<name>
```

Substitute `<name>` and expand `${CLAUDE_PROJECT_DIR}` to the real path before sending it. Add nothing else to the prompt.

## Step 3 — Report

Show the user both files, `claude/skills/<name>/SKILL.md` and `cursor/skills/<name>/SKILL.md`, along with the subagent's port report. Then stop.
