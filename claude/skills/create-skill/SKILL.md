---
name: create-skill
description: Create or revise a Claude Code skill — the SKILL.md directory, the frontmatter, and instructions that fire when they should. Use when the user wants a new skill written, an existing one fixed, or an explanation of the Claude Code skill format.
disable-model-invocation: true
---

# Create a Claude Code skill

A skill is a directory with a `SKILL.md` inside it: YAML frontmatter that tells Claude when the skill applies, and Markdown instructions Claude follows when it runs. Supporting files sit alongside it and load only when the body points at them.

The subject is whatever the user gave you: the skill argument, a description in the conversation, or an existing skill they want changed. If there is no subject, ask what the skill should do and stop.

## Step 1 — Settle what the skill is

A skill is a procedure, not a document. State it as one sentence of the form *when X happens, do Y*. If you cannot, it is not a skill — background knowledge belongs in `CLAUDE.md`, and a fact belongs in memory.

Settle these before writing anything:

- **The trigger.** The situation that should make this skill fire, in the user's words, not yours.
- **The one job.** Skills that do two unrelated things fire at the wrong time for both. Split them.
- **The end state.** What exists, or what has been said, when the skill is done.
- **The name.** Lowercase and hyphens. **The directory name is what the user types** — `.claude/skills/make-plan/SKILL.md` is `/make-plan`. Outside plugins the `name` field is display only.

## Step 2 — Put it in the right place

| Location   | Path                              | Applies to                    |
| :--------- | :-------------------------------- | :---------------------------- |
| Enterprise | Managed settings                  | All users in the organization |
| Personal   | `~/.claude/skills/<name>/SKILL.md`| All your projects             |
| Project    | `.claude/skills/<name>/SKILL.md`  | This project only             |
| Plugin     | `<plugin>/skills/<name>/SKILL.md` | Wherever the plugin is on     |

Default to project scope. Use personal scope only when the skill is about how *this user* works rather than about this codebase.

On a name clash: enterprise beats personal, personal beats project, and any of them beats a bundled skill of the same name. Plugin skills are namespaced `/plugin-name:skill-name` and never clash. Nested project skills that clash are namespaced by path, e.g. `apps/web:deploy`.

The directory layout:

```
my-skill/
├── SKILL.md          # required — frontmatter + instructions
├── reference.md      # optional — loaded only when SKILL.md points at it
└── scripts/
    └── helper.py     # optional — executed, not read into context
```

Only `SKILL.md` is required, and a skill that is just instructions is a complete skill.

## Step 3 — Write the frontmatter

Start minimal and add a field only when the skill actually needs it:

```yaml
---
name: make-plan
description: Turn a spec into a concrete implementation plan. Use whenever the user has a spec, feature request, or change description and wants a plan before any code is written.
---
```

**`description` is the whole routing decision.** It is the only part of the skill in context before invocation, and Claude picks the skill from it alone. Write both halves: what the skill does, then *when to use it*, in the vocabulary the user will actually type. A description that describes without naming a trigger produces a skill that never fires on its own. Put the key use case first — `description` plus `when_to_use` are truncated at 1,536 characters in the listing.

Then decide who may invoke it:

| Frontmatter                      | User invokes | Claude invokes | Context cost |
| :------------------------------- | :----------- | :------------- | :----------- |
| (default)                        | Yes          | Yes            | Description always in context; body loads on invoke |
| `disable-model-invocation: true` | Yes          | No             | Description not in context; body loads when invoked |
| `user-invocable: false`          | No           | Yes            | Description always in context; body loads on invoke |

Set `disable-model-invocation: true` for anything with side effects the user wants to time themselves (`/commit`, `/deploy`), and for reference-style skills that should never fire on their own. Set `user-invocable: false` for background knowledge that is not a meaningful command.

The rest of the fields — tools, model, effort, forked execution, hooks, arguments — are in [reference.md](reference.md). Read it before using any field not shown above.

## Step 4 — Write the body

The body is instructions to an agent, not documentation for a human.

- **Say what to do, not why.** Cut the rationale unless a rule fails without it.
- **Order the work.** `## Step 1 — …` headings when sequence matters; a flat checklist when it does not.
- **Make rules checkable.** "Never write files unless the user asks" beats "be careful about files".
- **State the stop condition.** Skills that do not say when they are finished tend to keep going.
- **Say what the output is** and where it goes — the conversation, a file, a diff.
- **Handle the missing input** in the first paragraph: what to do when the user invoked the skill with nothing.

Keep `SKILL.md` under ~500 lines. Once a skill loads, its content stays in context on every later turn, so every line is a recurring cost. Move long reference material into sibling files and point at them so they load only when needed:

```markdown
## Additional resources

- Full field reference: [reference.md](reference.md)
- Deploy helper: `${CLAUDE_SKILL_DIR}/scripts/deploy.sh <environment>`
```

Scripts are executed, not read into context, so a long script costs nothing until it runs.

## Step 5 — Check it before handing it over

- The description names a trigger, in words the user would use.
- The directory name is the command the user expects to type.
- The body would produce the right behaviour for someone who has never seen the conversation that created it.
- Every field used is a real one from [reference.md](reference.md), spelled with hyphens (except `when_to_use`).
- Nothing in the body duplicates what the sibling reference files already hold.

Show the user the file you wrote and the path it went to, then stop. Do not invoke the new skill to test it unless the user asks.
