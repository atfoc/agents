---
name: port-claude-skill-to-codex
description: "Ports or refreshes Claude Code skill bundles for Codex, preserving their workflow while adapting provider-specific metadata, inputs, paths, and tools. Use when converting a Claude skill to Codex or checking whether it can be reused unchanged."
---

# Port a Claude Code skill to Codex

Act on the skill names or source folders given in the request or conversation. In this repository, resolve a bare name to `claude/skills/<name>/`. If the source is missing, report it; do not author a replacement. If no source can be inferred, list relevant candidates and ask which to port. Handle multiple skills when requested.

## Choose the destination

Use the user's destination. Otherwise, in this repository write collection ports to `codex/skills/<name>/`. In other repositories, use the established convention or `.agents/skills/<name>/` for a project skill. Match the target folder name and frontmatter `name`.

`codex/skills/` is this repository's collection layout, not an automatic Codex discovery location. Project discovery uses `.agents/skills/`; user discovery uses `~/.agents/skills/`. Installing or linking a collection port is a separate action when requested. Do not change the installer CLI merely to create a port.

## Assess compatibility

Read `SKILL.md` and inventory its supporting files and external dependencies. Inspect referenced instructions, scripts, and configuration for Claude-specific assumptions; a compatible entrypoint can still call incompatible resources.

The shared Agent Skills format is a directory with YAML frontmatter, Markdown instructions, and optional resources. A skill containing only portable instructions and supported metadata can be copied unchanged to a discovered location. Loading successfully does not establish equivalent runtime behavior.

For provider-specific fields or syntax, read [the compatibility map](references/compatibility.md). Check current official documentation if the map does not cover a feature or the target runtime differs. Treat undocumented fields as unresolved rather than inventing an equivalent.

## Preserve the workflow

- Keep the Claude source intact. Use it as the authority for task behavior, inputs, outputs, constraints, authorization requirements, retry limits, and completion conditions.
- Retain prose and resource contents unless compatibility requires a change. Do not substitute a different service, model provider, or output format merely because Codex has a built-in alternative.
- Read an existing Codex port before refreshing it. Preserve working Codex adaptations that still satisfy the source. Inspect differences before overwriting; resolve ambiguous target changes with the user while completing unaffected work.
- Copy required sibling files, including helpers outside `scripts/`, licenses, executable permissions, and referenced assets. Exclude transient caches and build output only after checking whether the workflow needs them. Edit a supporting file when it also depends on Claude behavior; do not require every sibling to remain byte-identical.

## Adapt the bundle

1. Supply explicit, non-empty `name` and `description` strings. Use a lowercase hyphenated name of at most 64 characters and a description of at most 1,024 characters. Preserve useful trigger wording and merge relevant `when_to_use` guidance. Retain supported optional metadata; relocate or explain provider-specific fields using the map.
2. Replace Claude prompt substitutions with concrete input instructions. State which information comes from the request or prior conversation and what to do if required input is missing. Preserve argument ordering and defaults where they matter. Do not rewrite real shell variables or positional parameters as prompt syntax.
3. Express bundled resource links relative to the skill folder. For executed commands, tell Codex to resolve them against the loaded `SKILL.md` directory and pass quoted absolute paths, independent of the current working directory. Resolve project files against the actual project root. Do not hardcode this developer's installation path or invent `${CODEX_SKILL_DIR}`.
4. Replace Claude tool names with available Codex capabilities according to the operation. Preserve subagent isolation, waiting, scope, and review requirements when delegation is essential. When an essential capability is unavailable, retain an explicit blocker rather than silently changing the workflow.
5. Adapt references to other skills only when their Codex equivalents exist. Port a required dependency within the requested scope, or identify it as missing. Keep product names when they describe the actual product or service; replace them only when they mean the agent performing the work.
6. Add `agents/openai.yaml` only for useful UI metadata, invocation policy, or verified MCP dependencies. Preserve an existing file's unrelated fields. In particular, map a source's explicit-only invocation setting to `policy.allow_implicit_invocation: false`; leave the normal automatic default otherwise. Dependency metadata neither installs tools nor grants permission.

## Verify and report

- Parse the target YAML; check required fields, folder/name agreement, length limits, and supported target fields. Run the skill-creator validator when available. Parse `agents/openai.yaml` separately if present.
- Resolve resource links and command paths from the installed skill folder, including when the working directory is elsewhere. Check links to other skills and resources outside the bundle.
- Search the entire bundle for Claude substitutions, dynamic shell injection, tool names, and runtime fields. Review matches in context: examples about Claude and genuine shell syntax may remain; executable instructions must not depend on unimplemented substitution.
- Compare source and target file inventories and diffs. Account for each changed or omitted file and confirm the source was not modified. For changed helpers, use safe local checks appropriate to their behavior; do not invoke the ported workflow or call external services merely to validate the port.
- Review behavior using a representative request and a missing-input case. Confirm the same outcome, boundaries, and stop conditions. Label runtime behavior unverified when only static checks were possible.

Report whether the bundle was directly reusable or required adaptation, the created or updated path, meaningful conversions, validation results, and unresolved dependencies or behavior differences. Include a Codex invocation example such as `$<skill-name> <request>` and distinguish collection storage from an installed skill.

Finish when the requested ports are written and verified to the extent available, with remaining blockers disclosed. Stop there; do not run their actual workflows.
