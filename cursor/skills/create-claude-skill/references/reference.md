# Claude Code skill format — full reference

Read this when `SKILL.md` points you here: the complete frontmatter set, argument substitution,
dynamic context injection, and portability rules. This is Claude Code's format, not Cursor's —
none of it belongs in a Cursor skill file.

## Frontmatter reference

| Field                      | Required    | What it does |
| :------------------------- | :---------- | :----------- |
| `name`                     | No          | Display name in skill listings. Defaults to the directory name. |
| `description`              | Recommended | What the skill does and when to use it. Falls back to the first paragraph of the body. `description` + `when_to_use` are truncated at 1,536 characters in the listing — put the key use case first. |
| `when_to_use`              | No          | Extra trigger phrases or example requests, appended to `description` in the listing; counts toward the same 1,536-character cap. |
| `argument-hint`            | No          | Autocomplete hint, e.g. `[issue-number]` or `[filename] [format]`. |
| `arguments`                | No          | Named positional arguments for `$name` substitution. Space-separated string or YAML list; names map to positions in order. |
| `disable-model-invocation` | No          | `true` stops Claude from loading the skill on its own — you invoke it with `/name`. Also keeps it out of subagent preloading. Default `false`. Cursor has this same field, spelled the same way. |
| `user-invocable`           | No          | `false` hides it from the `/` menu; only Claude can invoke it. Default `true`. No Cursor equivalent. |
| `allowed-tools`            | No          | Tools pre-approved (no permission prompt) for the turn that invokes the skill. Space- or comma-separated string, or a YAML list. The grant clears on your next message. |
| `disallowed-tools`         | No          | Tools removed from the pool while the skill is active. Same formats; also clears on your next message. |
| `model`                    | No          | Model for the rest of the current turn. Same values as `/model`, or `inherit`. Not saved to settings. With `context: fork`, sets the forked subagent's model instead. |
| `effort`                   | No          | `low`, `medium`, `high`, `xhigh`, `max` while the skill is active. Defaults to the session level. |
| `context`                  | No          | `fork` runs the skill in a forked subagent context instead of the main conversation. |
| `agent`                    | No          | Which subagent type to use, when `context: fork` is set. |
| `background`               | No          | Only with `context: fork`. `false` waits for the forked subagent's result in the invoking turn instead of backgrounding it. Default `true`. |
| `hooks`                    | No          | Hooks registered when the skill is invoked; they keep running for the rest of the session. |
| `paths`                    | No          | Glob patterns limiting automatic activation to matching files. Comma-separated string or YAML list. Cursor has this same field, spelled the same way. |
| `shell`                    | No          | Shell for inline command injection: `bash` (default) or `powershell`. |
| `metadata`                 | No          | Free-form YAML map for your own tooling. Claude Code doesn't act on it; a non-map value is dropped. Don't reuse real field names as keys. Cursor has this same field, spelled the same way. |
| `license`                  | No          | Accepted, not acted on. |
| `compatibility`            | No          | Environment requirements, up to 500 characters. Accepted, not acted on. |

Boolean fields accept `true`/`false` plus `yes`, `no`, `on`, `off`, `1`, `0`, in any case.

Of the fields above, only `name`, `description`, `paths`, `disable-model-invocation`, and
`metadata` are also legal in a Cursor skill file. Everything else in this table only works when
the skill runs under Claude Code — using it in a `.cursor/skills/` file has no defined effect.

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

Cursor documents no equivalent syntax at all — a Cursor skill can never rely on a positional
argument arriving, so it has to read input loosely from the conversation instead. Declaring
`arguments` here is worth it only when the input reliably arrives as an argument; when it may
arrive as conversation instead, read it loosely in the body — "the spec is whatever the user
gave you: the skill argument, a file they pointed at, or the request in the conversation" — and
say what to do when it is missing.

## Dynamic context injection

A line like this in the body runs before Claude sees the skill, and is replaced by the output:

```markdown
## Current changes

!`git diff HEAD`
```

The instructions then arrive with the data already inlined. `shell` decides which shell runs it.
Cursor has no equivalent mechanism.

## Forked execution

`context: fork` runs the skill in a subagent with its own context window instead of in the main
conversation. Use it when the skill is expensive to run and only its conclusion matters. Cursor
has no equivalent — a Cursor skill always runs in the conversation (or subagent) that invoked it.

- `agent` picks which subagent type runs it.
- `model` then applies to that subagent, not the current turn.
- `background: false` makes the invoking turn wait for the result instead of backgrounding it.

## Portability

Claude Code skills follow the Agent Skills open standard and extend it. If a skill is also going
to be uploaded to claude.ai, used through the Skills API, or packaged with the standard packaging
script, only these fields travel with it: `name`, `description`, `license`, `compatibility`,
`metadata`, `allowed-tools`. Everything else above is a Claude Code extension and is unused
elsewhere.

## Things that trip people up

- **The description does the routing.** If it doesn't name the trigger ("Use whenever the user
  has a spec…"), the skill won't fire on its own.
- **The directory name is the command**, not the `name` field, outside of plugins — Cursor is
  stricter here and requires `name` to match the folder exactly.
- **`allowed-tools` is a permission grant, not a restriction** — use `disallowed-tools` to take
  tools away. Cursor has no permission-grant mechanism at all; a Cursor skill always prompts.
- **Both tool fields clear on the next user message.** They cover the invoking turn, not the
  rest of the session.
- **Field names are lowercase with hyphens** (`allowed-tools`, `argument-hint`,
  `disable-model-invocation`) — except `when_to_use`, which uses an underscore.
- **Loaded skill content is a recurring cost**, charged every turn after it loads, not once.

## Cross-tool portability

Cursor reads `.claude/skills/` as a compatibility path but supports only `name`, `description`,
`paths`, `disable-model-invocation` and `metadata`. A skill meant to work in both places should
keep to those five fields and put everything else in the body. The `create-cursor-skill` skill
documents that side of the pair in full.
