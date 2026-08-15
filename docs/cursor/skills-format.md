# Cursor — skill format

A skill is a portable, version-controlled folder that teaches the agent how to do one
domain-specific task. It is a directory containing a `SKILL.md`: YAML frontmatter saying what
the skill is and when it applies, plus Markdown instructions the agent follows. Scripts,
templates, and reference documents can sit beside it.

The skills in this repo live in `cursor/skills/` (`make-plan/SKILL.md`,
`implement-plan/SKILL.md`) and are installed into a project's `.cursor/skills/` by
`cursor/install-cursor-config.sh`.

## Directory layout

```
my-skill/
├── SKILL.md        # required — frontmatter + instructions
├── scripts/        # optional — executable code the agent can run
├── references/     # optional — extra docs loaded on demand
└── assets/         # optional — templates, images, data files
```

Only `SKILL.md` is required. A skill that is just instructions is a complete skill — that is
what both skills in this repo are.

## Where skills live

Loaded automatically from:

- `.cursor/skills/` (project)
- `.agents/skills/` (project)
- `~/.cursor/skills/` (global)
- `~/.agents/skills/` (global)

Compatibility paths, also read: `.claude/skills/`, `.codex/skills/`, and their `~/` equivalents.

Skills nest recursively. A skill in a nested project directory — `apps/web/.cursor/skills/` in
a monorepo — is automatically scoped to files inside that directory, so directory-specific
guidance stays out of context for unrelated work. That scoping is free: it needs no `paths`
entry.

Inside a plugin, skills are subdirectories of the plugin's `skills/` directory, each with a
`SKILL.md`, discovered by folder scan.

## Minimal file

```markdown
---
name: make-plan
description: Turn a spec into a concrete implementation plan. Use whenever the user has a spec, feature request, or change description and wants a plan before any code is written.
---

# Make a plan

Turn a spec into a plan specific enough that someone else could implement it without
re-deciding anything.

## Step 1 — ...
```

## Frontmatter reference

| Field                      | Required | Type            | What it does |
| :------------------------- | :------- | :-------------- | :----------- |
| `name`                     | Yes      | string          | Lowercase letters, numbers, and hyphens only. Must match the parent folder name. |
| `description`              | Yes      | string          | What the skill does and when to use it. This is what the agent reads when deciding whether the skill is relevant. |
| `paths`                    | No       | string or list  | Glob patterns limiting the skill to matching files. Comma-separated string or YAML list. Leave unset for a skill that should be available regardless of which files are open. |
| `disable-model-invocation` | No       | boolean         | `true` means the skill is only included when explicitly invoked as `/skill-name`. |
| `metadata`                 | No       | map             | Arbitrary key-value data. |

`globs` is a legacy alias still accepted in place of `paths`; use `paths` in new skills.

## How skills get invoked

- **Automatically.** The agent is shown the available skills and decides when one is relevant
  from the conversation — unless `disable-model-invocation: true`.
- **Manually.** Type `/` in Agent chat and search for the skill by name.
- **By scope.** A skill in a nested project directory applies to work inside that directory.

Cursor documents no formal syntax for passing arguments to a skill. Write skills so they read
their input from the conversation and from what the user pointed at, rather than assuming a
positional argument arrives. Both skills in this repo do that — "the plan is whatever the user
pointed at: the plan already in this conversation, a file, or the skill argument" — and both say
what to do when the input is missing.

## Supporting files

`SKILL.md` stays the entry point; everything else loads only when the agent needs it. Reference
the extras explicitly so the agent knows what each one holds and when to open it:

```markdown
## Additional resources

- Full API details: `references/api.md`
- Deploy helper: `scripts/deploy.sh <environment>`
```

Scripts are run with the agent's tools, not read into context, so a long script costs nothing
until it executes.

## Skills, rules, and commands

Skills are the current form of packaged, reusable instructions in Cursor. Cursor's
`/migrate-to-skills` conversion turns dynamic rules and slash commands into skills, with
commands becoming skills marked `disable-model-invocation: true` — that pairing is the rule of
thumb: if it is something the user triggers deliberately, mark it so the agent won't fire it on
its own. Rules with `alwaysApply: true` or explicit `globs` are not migrated, because their
triggering conditions differ from how skills activate.

## What this format does not give you

Cursor documents exactly the five frontmatter fields above. A skill file cannot configure the
model, effort, tool permissions, argument names, autocomplete hints, hooks, or subagent
execution — none of those keys are part of the format, and Cursor's docs don't state what
happens to unrecognised ones. Keep frontmatter to the documented fields and express everything
else as instructions in the body.

Two consequences worth planning around:

- **No tool pre-approval.** A skill cannot grant itself permission to run tools without asking;
  permissions stay wherever the user set them.
- **Subagent nesting stops after one level.** A skill invoked inside a subagent cannot fan work
  out to further subagents, so any skill that describes delegation should say what the agent
  does instead when it is already running as a subagent.

## Things that trip people up

- **`name` must match the folder name**, and the folder name is what you type as `/name`.
- **The description does the routing.** If it doesn't name the trigger ("Use whenever the user
  has a spec…"), the skill will not fire on its own.
- **Skill content is a recurring context cost** once loaded. Keep `SKILL.md` to the essentials
  and push long reference material into `references/`.
