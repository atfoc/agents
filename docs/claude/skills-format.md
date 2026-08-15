# Claude Code — skill format

A skill is a directory with a `SKILL.md` inside it: YAML frontmatter that tells Claude when to
use the skill, and Markdown instructions it follows when the skill runs. Supporting files can
sit alongside it and are loaded only when needed.

The skills in this repo live in `claude/skills/` (`make-plan/SKILL.md`,
`implement-plan/SKILL.md`, `research/SKILL.md`) and are installed into a project's
`.claude/skills/`.

## Directory layout

```
my-skill/
├── SKILL.md          # required — frontmatter + instructions
├── reference.md      # optional — loaded only when SKILL.md points at it
├── examples.md       # optional
└── scripts/
    └── helper.py     # optional — executed, not read into context
```

Only `SKILL.md` is required. A skill with nothing else is a normal, complete skill — that is
what every skill in this repo is.

## Where skills live

| Location   | Path                                       | Applies to                     |
| :--------- | :----------------------------------------- | :----------------------------- |
| Enterprise | Managed settings                           | All users in the organization  |
| Personal   | `~/.claude/skills/<name>/SKILL.md`         | All your projects              |
| Project    | `.claude/skills/<name>/SKILL.md`           | This project only              |
| Plugin     | `<plugin>/skills/<name>/SKILL.md`          | Wherever the plugin is on      |

On a name clash: enterprise beats personal, personal beats project, and any of them beats a
bundled skill of the same name. Plugin skills are namespaced as `/plugin-name:skill-name` and
so never clash. Files in `.claude/commands/` behave the same way, and a skill wins over a
command of the same name.

The directory name is what you type. `.claude/skills/make-plan/SKILL.md` is `/make-plan`.
Nested project skills that clash are namespaced by path, e.g. `apps/web:deploy`.

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

All fields are optional; only `description` is really recommended, since it is what Claude
reads when deciding whether the skill applies. Boolean fields accept `true`/`false` plus
`yes`, `no`, `on`, `off`, `1`, `0`, in any case.

## Frontmatter reference

| Field                      | Required    | What it does |
| :------------------------- | :---------- | :----------- |
| `name`                     | No          | Display name in skill listings. Defaults to the directory name. |
| `description`              | Recommended | What the skill does and when to use it. Falls back to the first paragraph of the body. `description` + `when_to_use` are truncated at 1,536 characters in the listing — put the key use case first. |
| `when_to_use`              | No          | Extra trigger phrases or example requests, appended to `description` in the listing; counts toward the same 1,536-character cap. |
| `argument-hint`            | No          | Autocomplete hint, e.g. `[issue-number]` or `[filename] [format]`. |
| `arguments`                | No          | Named positional arguments for `$name` substitution. Space-separated string or YAML list; names map to positions in order. |
| `disable-model-invocation` | No          | `true` stops Claude from loading the skill on its own — you invoke it with `/name`. Also keeps it out of subagent preloading. Default `false`. |
| `user-invocable`           | No          | `false` hides it from the `/` menu; only Claude can invoke it. Default `true`. |
| `allowed-tools`            | No          | Tools pre-approved (no permission prompt) for the turn that invokes the skill. Space- or comma-separated string, or a YAML list. The grant clears on your next message. |
| `disallowed-tools`         | No          | Tools removed from the pool while the skill is active. Same formats; also clears on your next message. |
| `model`                    | No          | Model for the rest of the current turn. Same values as `/model`, or `inherit`. Not saved to settings. With `context: fork`, sets the forked subagent's model instead. |
| `effort`                   | No          | `low`, `medium`, `high`, `xhigh`, `max` while the skill is active. Defaults to the session level. |
| `context`                  | No          | `fork` runs the skill in a forked subagent context instead of the main conversation. |
| `agent`                    | No          | Which subagent type to use, when `context: fork` is set. |
| `background`               | No          | Only with `context: fork`. `false` waits for the forked subagent's result in the invoking turn instead of backgrounding it. Default `true`. |
| `hooks`                    | No          | Hooks registered when the skill is invoked; they keep running for the rest of the session. |
| `paths`                    | No          | Glob patterns limiting automatic activation to matching files. Comma-separated string or YAML list. |
| `shell`                    | No          | Shell for inline command injection: `bash` (default) or `powershell`. |
| `metadata`                 | No          | Free-form YAML map for your own tooling. Claude Code doesn't act on it; a non-map value is dropped. Don't reuse real field names as keys. |
| `license`                  | No          | Accepted, not acted on. |
| `compatibility`            | No          | Environment requirements, up to 500 characters. Accepted, not acted on. |

## Who can invoke what

| Frontmatter                      | You can invoke | Claude can invoke | Context cost |
| :------------------------------- | :------------- | :---------------- | :----------- |
| (default)                        | Yes            | Yes               | Description always in context; body loads on invoke |
| `disable-model-invocation: true` | Yes            | No                | Description not in context; body loads when you invoke |
| `user-invocable: false`          | No             | Yes               | Description always in context; body loads on invoke |

Use `disable-model-invocation: true` for anything with side effects you want to time yourself
(`/commit`, `/deploy`). Use `user-invocable: false` for background knowledge that isn't a
meaningful command.

## Arguments and substitutions

Available inside the skill body:

| Placeholder             | Expands to |
| :---------------------- | :--------- |
| `$ARGUMENTS`            | Everything passed to the skill. If absent from the body, arguments are appended as `ARGUMENTS: <value>`. |
| `$ARGUMENTS[N]` / `$N`  | One argument by 0-based index. |
| `$name`                 | A named argument declared in `arguments`. |
| `${CLAUDE_SKILL_DIR}`   | The directory holding this `SKILL.md` — use it to reference bundled scripts. |
| `${CLAUDE_PROJECT_DIR}` | The project root. |
| `${CLAUDE_SESSION_ID}`  | Current session ID. |
| `${CLAUDE_EFFORT}`      | Current effort level. |
| `${CLAUDE_PLUGIN_ROOT}` | Plugin install directory (plugin skills only). |
| `${CLAUDE_PLUGIN_DATA}` | Plugin persistent data directory (plugin skills only). |

The skills in this repo read their input loosely instead — "the plan is whatever the user
pointed at: the plan already in this conversation, a file, or the skill argument" — which is
the right call when the input may arrive as conversation rather than as an argument.

## Dynamic context injection

A line like this in the body runs before Claude sees the skill, and is replaced by the output:

```markdown
## Current changes

!`git diff HEAD`
```

The instructions then arrive with the data already inlined. `shell` decides which shell runs it.

## Keeping skills cheap

Once a skill loads, its content stays in context across turns, so every line is a recurring
cost. Keep `SKILL.md` under ~500 lines and state what to do rather than narrating why. Move
long reference material into sibling files and point at them so they load only when needed:

```markdown
## Additional resources

- For complete API details, see [reference.md](reference.md)
- For usage examples, see [examples.md](examples.md)
```

## Portability

Claude Code skills follow the Agent Skills open standard and extend it. If a skill is also
going to be uploaded to claude.ai, used through the Skills API, or packaged with the standard
packaging script, only these fields travel with it: `name`, `description`, `license`,
`compatibility`, `metadata`, `allowed-tools`. Everything else in the table above is a Claude
Code extension and is simply unused elsewhere.

## Things that trip people up

- **The description does the routing.** If it doesn't name the trigger ("Use whenever the user
  has a spec…"), the skill won't fire on its own.
- **The directory name is the command**, not the `name` field, outside of plugins.
- **`allowed-tools` is a permission grant, not a restriction** — use `disallowed-tools` to take
  tools away.
- **Field names are lowercase with hyphens** (`allowed-tools`, `argument-hint`,
  `disable-model-invocation`) — except `when_to_use`, which uses an underscore.
