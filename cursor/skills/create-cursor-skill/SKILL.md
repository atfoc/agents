---
name: create-cursor-skill
description: Create or revise a Cursor skill — the SKILL.md folder, its frontmatter, and instructions that fire when they should. Use when the user wants a new Cursor skill written, an existing one fixed, or an explanation of the Cursor skill format. Not for Claude Code skills — use create-claude-skill for those.
disable-model-invocation: true
---

# Create a Cursor skill

A skill is a portable, version-controlled folder that teaches the agent one domain-specific task. It is a directory containing a `SKILL.md`: YAML frontmatter saying what the skill is and when it applies, plus Markdown instructions the agent follows. Scripts, templates and reference documents can sit beside it.

The subject is whatever the user gave you: the skill argument, a description in the conversation, or an existing skill they want changed. If there is no subject, ask what the skill should do and stop.

## Step 1 — Settle what the skill is

A skill is a procedure, not a document. State it as one sentence of the form *when X happens, do Y*. If you cannot, it is not a skill — project-wide background belongs in a rule, and a fact belongs in the project docs.

Settle these before writing anything:

- **The trigger.** The situation that should make this skill fire, in the user's words, not yours.
- **The one job.** Skills that do two unrelated things fire at the wrong time for both. Split them.
- **The end state.** What exists, or what has been said, when the skill is done.
- **The name.** Lowercase letters, numbers and hyphens only, and **it must match the folder name** — the folder name is also what the user types as `/name`.

## Step 2 — Put it in the right place

Skills are loaded automatically from:

- `.cursor/skills/` (project)
- `.agents/skills/` (project)
- `~/.cursor/skills/` (global)
- `~/.agents/skills/` (global)

`.claude/skills/` and `.codex/skills/`, plus their `~/` equivalents, are read as compatibility paths. Inside a plugin, skills are subdirectories of the plugin's `skills/` directory, each with a `SKILL.md`, discovered by folder scan.

Default to project scope. Use a global path only when the skill is about how *this user* works rather than about this codebase.

Skills nest recursively, and a skill in a nested project directory — `apps/web/.cursor/skills/` in a monorepo — is automatically scoped to files inside that directory. That scoping is free: it needs no `paths` entry. Use it instead of `paths` whenever the skill belongs to one part of the tree.

The directory layout:

```
my-skill/
├── SKILL.md        # required — frontmatter + instructions
├── scripts/        # optional — executable code the agent can run
├── references/     # optional — extra docs loaded on demand
└── assets/         # optional — templates, images, data files
```

Only `SKILL.md` is required, and a skill that is just instructions is a complete skill.

## Step 3 — Write the frontmatter

Cursor documents exactly five fields. Use them and nothing else:

| Field                      | Required | Type           | What it does |
| :------------------------- | :------- | :------------- | :----------- |
| `name`                     | Yes      | string         | Lowercase letters, numbers and hyphens only. Must match the parent folder name. |
| `description`              | Yes      | string         | What the skill does and when to use it. This is what the agent reads when deciding whether the skill is relevant. |
| `paths`                    | No       | string or list | Glob patterns limiting the skill to matching files. Comma-separated string or YAML list. Leave unset for a skill that should be available regardless of which files are open. |
| `disable-model-invocation` | No       | boolean        | `true` means the skill is only included when explicitly invoked as `/skill-name`. |
| `metadata`                 | No       | map            | Arbitrary key-value data. |

`globs` is a legacy alias still accepted in place of `paths`; use `paths` in new skills.

A minimal file:

```yaml
---
name: make-plan
description: Turn a spec into a concrete implementation plan. Use whenever the user has a spec, feature request, or change description and wants a plan before any code is written.
---
```

**`description` is the whole routing decision.** It is the only part of the skill the agent sees before invoking it. Write both halves: what the skill does, then *when to use it*, in the vocabulary the user will actually type. A description that describes without naming a trigger produces a skill that never fires on its own.

Set `disable-model-invocation: true` when the skill is something the user triggers deliberately — anything with side effects (`/commit`, `/deploy`), and reference-style skills that should never fire on their own. This is the same pairing Cursor's own `/migrate-to-skills` conversion uses when it turns slash commands into skills.

## Step 4 — Write the body

The body is instructions to an agent, not documentation for a human.

- **Say what to do, not why.** Cut the rationale unless a rule fails without it.
- **Order the work.** `## Step 1 — …` headings when sequence matters; a flat checklist when it does not.
- **Make rules checkable.** "Never write files unless the user asks" beats "be careful about files".
- **State the stop condition.** Skills that do not say when they are finished tend to keep going.
- **Say what the output is** and where it goes — the conversation, a file, a diff.
- **Read the input loosely.** Cursor documents no syntax for passing arguments to a skill, so never assume a positional argument arrives. Write "the spec is whatever the user pointed at: the request in this conversation, a file, or the skill argument", and say what to do when it is missing.
- **Handle the subagent case.** Subagent nesting stops after one level, so a skill invoked inside a subagent cannot fan work out to further subagents. Any skill that describes delegation must say what the agent does instead when it is already running as a subagent.

Keep `SKILL.md` to the essentials. Loaded skill content is a recurring context cost, so push long reference material into `references/` and point at it explicitly, with what each file holds and when to open it:

```markdown
## Additional resources

- Full API details: `references/api.md`
- Deploy helper: `scripts/deploy.sh <environment>`
```

Scripts are run with the agent's tools, not read into context, so a long script costs nothing until it executes.

## Step 5 — Know what the format will not do for you

Cursor documents only those five frontmatter fields. A skill file cannot configure the model, effort, tool permissions, argument names, autocomplete hints, hooks, or subagent execution — none of those keys are part of the format, and Cursor's docs do not state what happens to unrecognised ones. Keep frontmatter to the documented fields and express everything else as instructions in the body.

Two consequences worth planning around:

- **No tool pre-approval.** A skill cannot grant itself permission to run tools without asking; permissions stay wherever the user set them. A skill that assumes silent tool access will stall on a prompt.
- **No enforced restrictions either.** "Never delete anything" is honoured by instruction, not by the harness, so state such rules plainly and repeat them where they matter.

## Step 6 — Check it before handing it over

- `name` matches the folder name, and the folder name is the command the user expects to type.
- The description names a trigger, in words the user would use.
- Only the five documented fields appear in the frontmatter.
- The body says what to do when the input is missing, and what to do when running as a subagent.
- The body would produce the right behaviour for someone who has never seen the conversation that created it.

Show the user the file you wrote and the path it went to, then stop. Do not invoke the new skill to test it unless the user asks.

Building this same skill for Claude Code instead? Use the `create-claude-skill` skill — Claude Code documents many more frontmatter fields, plus arguments, forked execution, and dynamic context injection that Cursor has no equivalent for.
