# Claude Code to Codex compatibility map

Checked against official documentation on 2026-10-01. These are migration decisions for standalone skills, not a claim that every client implements all Agent Skills extensions. Recheck the target runtime when a feature is essential or not listed here.

## File format and discovery

Both products use `SKILL.md` with YAML frontmatter and a Markdown body. Supporting scripts, references, and assets share the same directory model. Codex requires explicit `name` and `description`; Claude Code can derive them when omitted. Use the stricter shared specification for a portable bundle. [Agent Skills specification](https://agentskills.io/specification), [Claude Code skills](https://code.claude.com/docs/en/skills).

For local discovery, move or link the complete folder from Claude's `.claude/skills/` to Codex's `.agents/skills/`. Collection folders such as this repository's `claude/skills/` and `codex/skills/` are source storage. Codex can discover symlinked skill folders. Prefer current documented locations over old `.codex/skills` examples. [Codex skill discovery](https://learn.chatgpt.com/docs/build-skills#where-codex-loads-local-skills).

## Frontmatter decisions

This table gives conservative mappings based on the [Claude fields](https://code.claude.com/docs/en/skills#frontmatter-reference) and [Codex configuration](https://learn.chatgpt.com/docs/build-skills).

| Source field | Porting decision |
| --- | --- |
| `name`, `description` | Keep valid values; supply missing ones and satisfy target limits. |
| `when_to_use` | Fold useful triggers into `description`; avoid duplicating the body. |
| `license`, `metadata` | Preserve supported values and license resources. |
| `compatibility` | Defined by the shared standard; verify target support. If unavailable, state requirements in the body. |
| `argument-hint`, `arguments` | State the input contract in prose; optional UI prompts do not implement argument parsing. |
| `disable-model-invocation: true` | Remove this Claude field; set `policy.allow_implicit_invocation: false` in `agents/openai.yaml`. This maps explicit-only discovery intent, not every Claude scheduling rule. |
| `user-invocable: false` | No documented standalone Codex equivalent for model-only visibility. Disclose that difference; implicit invocation policy is not a substitute. |
| `allowed-tools` | Experimental in the standard. Do not assume Claude tool patterns or per-turn approval grants carry over. Preserve necessary operations in the body; use actual Codex permissions. |
| `disallowed-tools` | Do not claim a prose restriction enforces a tool ban. Preserve the behavioral constraint and disclose any missing enforcement. |
| `model`, `effort` | Remove Claude overrides. Use target session configuration or supported delegation settings when requested; do not infer a model mapping. |
| `context`, `agent`, `background` | Replace runtime fields with explicit delegation instructions when supported. Preserve required isolation and waiting; otherwise disclose the gap. |
| `hooks` | No direct skill-frontmatter conversion. Translate required behavior to explicit steps or separately adapt supported Codex hooks. |
| Unknown fields | Check target documentation; preserve intent in supported instructions or report the unresolved behavior. |

`allowed-tools` in Claude grants approval for listed tools; it is not an exclusive tool whitelist. Neither parsing that field nor copying it proves Codex grants equivalent permission.

Codex invocation policy belongs in a separate file, not `SKILL.md`:

```yaml
# agents/openai.yaml, only for a source that requires explicit invocation
interface:
  display_name: "Review Changes"
  short_description: "Review changes against project requirements"
policy:
  allow_implicit_invocation: false
```

For an ordinary skill, omit that policy or use `true`. When adding an optional `default_prompt`, include `$<skill-name>` in a useful example request. Do not regenerate existing metadata in a way that drops dependencies or policy.

## Runtime adaptation

Translate the source's execution contract rather than performing a global textual replacement:

- For `$ARGUMENTS`, indexed forms, or declared named arguments, describe the actual input and how it is obtained. Preserve real script arguments such as shell `$1`.
- For `${CLAUDE_SKILL_DIR}`, resolve bundled files relative to the loaded skill directory. For project, plugin, session, effort, and persistent-data variables, identify the needed value and its actual target source; do not invent environment variables.
- For Claude's `!` followed by a backtick command, turn required context gathering into an explicit tool step with error handling. Remove reliance on preprocessing; do not execute these commands while porting.
- For `/other-skill` or the Claude Skill tool, identify the available Codex skill and load its instructions. A textual `$other-skill` reference alone does not prove the dependency exists.
- For `Read`, `Write`, `Edit`, `Grep`, and `Bash`, specify the file or command operation using available tools. For `AskUserQuestion`, use a supported input tool or a conversation question. For `TodoWrite`, preserve task tracking without requiring that exact tool.
- For the Claude Agent tool or named agents, inspect the role's requirements. Use available Codex delegation only when supported and authorized; otherwise retain a blocker for essential isolation. Do not equate Codex UI metadata in `agents/openai.yaml` with Claude agent definitions.

## When a standalone port is insufficient

A skill that requires plugin settings, hooks, MCP servers, or Claude artifacts may need additional adaptation. Keep essential dependencies explicit; a skill folder does not configure a server or install a plugin. Codex has its own hook runtime, so avoid claiming all hooks are unsupported. For plugin conversion, consult [OpenAI's Claude plugin migration guide](https://developers.openai.com/plugins/guides/submit-claude-plugin). Do not expand a requested skill port into publication or plugin installation.
