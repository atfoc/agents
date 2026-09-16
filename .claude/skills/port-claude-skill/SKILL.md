---
name: port-claude-skill
description: Ports one Claude Code skill to Cursor by converting its SKILL.md to the Cursor skill format and copying its sibling files unchanged. Use when a skill under claude/skills/ needs to be created or refreshed under cursor/skills/.
argument-hint: [skill name]
---

# Port a Claude Code skill to Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or a skill named earlier in the conversation. It is a bare skill name or a path to a skill folder under `claude/skills/`. Port exactly one skill. If no skill is named, list the skills under `claude/skills/` that have no counterpart under `cursor/skills/`, ask which to port, and stop. If more than one is named, ask which one to port and stop.

A skill named `<name>` lives at `claude/skills/<name>/` and ports to `cursor/skills/<name>/`.

## Rules that hold throughout

- **The Claude source is the authority.** Where an existing Cursor version disagrees with it, the Cursor version loses, including wording someone tuned by hand there. Read an existing Cursor version only to see what is about to change — never copy wording from it.
- **Preserve the source's prose.** No rewording, no reordering, no tightening. Only the three classes of change below are permitted. The description is carried over verbatim.
- **Skills only.** Agents are not skills; never port one this way.
- **Never author a missing source skill.** If the named skill does not exist under `claude/skills/`, report that as an error rather than writing one.
- **Do not use other Cursor skills as a template.** The format rules below are the whole of what the Cursor side needs.

## What the port consists of

The Cursor skill folder is the Claude skill folder, copied whole, with only `SKILL.md` rewritten. Sibling files — `scripts/`, `references/`, `assets/` and anything else in the folder — are copied byte for byte and never edited. When a Cursor version already exists, replace the folder entirely so no stale files survive.

## The three permitted changes to SKILL.md

### 1. Mechanical conversions

- `$ARGUMENTS` → wording that reads the input loosely: "the skill argument".
- `${CLAUDE_SKILL_DIR}/<path>` → the relative path `<path>`.
- `${CLAUDE_PROJECT_DIR}` → the repo-relative path.
- "Claude" or "Claude Code", where it names the agent doing the work → "the agent". Leave it alone where it names the product or a path: `claude/skills/`, `create-claude-skill`, "Claude Code documents many more frontmatter fields".
- Frontmatter: keep only `name`, `description`, `paths`, `disable-model-invocation` and `metadata`. Drop everything else — `argument-hint`, `allowed-tools`, `model`, `context`, and any other field. `name` must match the folder name.

### 2. Removals

Delete instructions that are meaningless or broken under Cursor. Nothing else.

### 3. Additions

Ask these three questions of the source body. Add a paragraph only where the answer is no.

1. **Does it say what the skill acts on, and what to do when that is missing?** Both halves, not one. A skill that says "the subject is whatever the user gave you… if there is no description, ask for one and stop" needs nothing. A skill that names its input but never says what happens when it is absent gets that paragraph, placed first after the `#` heading.
2. **Does it say when the skill is finished?** If not, add a stop condition as the last line of the body.
3. **Does it describe delegating to subagents?** If it never mentions delegation, add nothing. If it delegates but never says what happens when it is already running inside a subagent, add that caveat — nesting stops after one level — immediately after the passage describing the delegation.

Add nothing else. In particular, do not add "do not use subagents" to a skill whose source never mentions them.

## Checking the result

A port is done when all of these hold for the Cursor folder:

- No file contains `$ARGUMENTS` or `${CLAUDE_`.
- `SKILL.md` opens with a frontmatter block containing only the permitted fields, its `name` equals the folder name, and its `description` is identical to the source's.
- Every `scripts/`, `references/` or `assets/` path the body mentions exists in the folder.
- Diffing against the Claude source shows only the three classes of change above.

Report whether the Cursor version was created or updated, which frontmatter fields were dropped, which substitutions and additions were made, which sibling files were copied, and — for an update — anything that changed versus the previous Cursor version. Then stop; do not invoke the ported skill to test it.
