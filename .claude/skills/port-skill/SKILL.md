---
name: port-skill
description: Port an existing skill from claude/skills/ to cursor/skills/ — copy it, convert what Cursor does not support, verify, and report. Use when a skill exists on the Claude side and is missing or stale on the Cursor side.
argument-hint: [skill name(s)]
disable-model-invocation: true
---

# Port a skill from Claude Code to Cursor

The subject is whatever the user gave you: `$ARGUMENTS`, or skill names named earlier in the conversation. Each is a bare skill name or a path under `claude/skills/`.

`<ROOT>` below is the absolute path to this repository. Expand it to the real path in every command and every subagent prompt — never send the literal `${CLAUDE_PROJECT_DIR}` to a subagent.

## Rules that hold throughout

- **The Claude source is the authority.** Where the Cursor version disagrees with it, the Cursor version loses, including wording someone tuned by hand there.
- **Preserve the source's prose.** No rewording, no reordering, no tightening. Only the four change classes in Step 3 are permitted.
- **Skills only.** Never port an agent, never write anything under `cursor/agents/`.
- **Never author a missing source skill.** If a named skill does not exist under `claude/skills/`, that is an error to report, not a gap to fill. Authoring is `create-skill`'s job.
- **The Cursor format authority is `<ROOT>/claude/skills/create-cursor-skill/SKILL.md`.** Do not open other skills in `cursor/skills/` as a template or a source of wording.

## Step 1 — Resolve what to port

List what exists on each side:

```bash
python3 -c '
import os,sys
root=sys.argv[1]
def skills(kind):
    d=os.path.join(root,kind,"skills")
    return {n for n in os.listdir(d) if os.path.isfile(os.path.join(d,n,"SKILL.md"))} if os.path.isdir(d) else set()
src,dst=skills("claude"),skills("cursor")
print("MISSING: "+(", ".join(sorted(src-dst)) or "(none)"))
print("PORTED : "+(", ".join(sorted(src&dst)) or "(none)"))
' <ROOT>
```

**No skill named.** Show the `MISSING` list as the skills waiting to be ported, add one line saying any skill in `PORTED` can also be named explicitly to re-port it, then stop and wait. Do not pick one. Do not port everything missing.

**Names given.** For each, in order:

1. **Refuse agents.** If the raw input the user typed contains `agents/`, stop: "this skill ports skills only; `<raw>` is an agent." Do not check anything else for that name.
2. **Normalise.** Strip a trailing `SKILL.md`, strip trailing slashes, take the final path component. `research`, `claude/skills/research`, `claude/skills/research/`, `./claude/skills/research/SKILL.md` and their absolute forms all reduce to `research`.
3. **Validate.** The name must appear in `MISSING` or `PORTED` from the command above. If it does not, that is an error. When `<ROOT>/claude/agents/<name>.md` exists, append one line to the error: "`<name>` is an agent, and this skill ports skills only."

**Gate.** Do not proceed to Step 2 until every named skill has passed all three. If any name fails, port nothing at all — report every bad name, list what `claude/skills/` actually contains, and stop.

## Step 2 — Choose the execution mode

**One skill.** Port it inline: go to Step 3 yourself.

**More than one skill, and you are not a subagent.** Spawn one subagent per skill with the Agent tool, passing no `subagent_type`, all in a single message so they run in parallel, then wait for all of them. Each prompt is exactly:

```
Follow Step 3 onward of <ROOT>/.claude/skills/port-skill/SKILL.md to port exactly one skill.

Repo root: <ROOT>
Skill name: <name>

Port only that skill. Do not spawn further subagents.
Report back the Step 5 report block for this skill and nothing else.
```

Do not paste the skill's content, the checklist, or a summary into the prompt — the subagent reads both files itself.

When they return, concatenate their blocks verbatim under one heading and add nothing beyond a one-line count. Do not re-verify their work and do not rewrite their blocks. A subagent that fails gets a `FAILED` block naming what went wrong; the skills that succeeded stay ported and are not rolled back.

**More than one skill, and you are already running as a subagent.** Stop before porting anything: subagent nesting stops after one level, so the per-skill subagents cannot be spawned. Tell the user to re-run this from the main session, or to invoke it once per skill. Port nothing — no partial work. A single skill inside a subagent is unaffected: it ports inline.

## Step 3 — Port one skill

Read, in this order:

1. `<ROOT>/claude/skills/<name>/SKILL.md` — the file being ported.
2. `<ROOT>/claude/skills/create-cursor-skill/SKILL.md` — the Cursor format, plus any file it points you to.
3. `<ROOT>/cursor/skills/<name>/SKILL.md`, **only if it already exists**, and only to see what is about to change for the report. Never copy wording from it.

Then stage the destination:

```bash
python3 -c '
import os,shutil,sys
root,name=os.path.realpath(sys.argv[1]),sys.argv[2]
if os.sep in name or name in (".","..",""): sys.exit("bad name: "+name)
if not os.path.isdir(os.path.join(root,"cursor")): sys.exit("not a repo root: "+root)
src=os.path.join(root,"claude","skills",name)
if not os.path.isfile(os.path.join(src,"SKILL.md")): sys.exit("no such skill: "+name)
base=os.path.join(root,"cursor","skills")
os.makedirs(base,exist_ok=True)
dst=os.path.realpath(os.path.join(base,name))
if not dst.startswith(os.path.realpath(base)+os.sep): sys.exit("refusing: destination escapes cursor/skills")
existed=os.path.isdir(dst)
if existed: shutil.rmtree(dst)
shutil.copytree(src,dst,symlinks=False)
sibs=sorted(os.path.relpath(os.path.join(dp,f),dst) for dp,_,fs in os.walk(dst) for f in fs if os.path.relpath(os.path.join(dp,f),dst)!="SKILL.md")
print(("UPDATED " if existed else "CREATED ")+dst)
print("siblings: "+(", ".join(sibs) or "(none)"))
' <ROOT> <name>
```

It prints `CREATED` or `UPDATED` and the sibling files — both go straight into the Step 5 report. The destination now holds a byte-for-byte copy of the Claude skill, `SKILL.md` included. `scripts/`, `references/` and `assets/` are already correct and must not be edited.

Now write the ported `SKILL.md` over the copied one. Four classes of change are permitted and nothing else.

### 1. The mechanical conversions

- `$ARGUMENTS` → wording that reads the input loosely: "the skill argument"
- `${CLAUDE_SKILL_DIR}/<path>` → the relative path `<path>`
- `${CLAUDE_PROJECT_DIR}` → the repo-relative path
- "Claude" or "Claude Code", where it names the agent doing the work → "the agent". Leave it alone where it names the product or a path: `claude/skills/`, `create-claude-skill`, "Claude Code documents many more frontmatter fields".
- Frontmatter: keep only `name`, `description`, `paths`, `disable-model-invocation`, `metadata`. Drop everything else — `argument-hint`, `allowed-tools`, `model`, `context`, and any field not in that list.

### 2. Removal

Delete instructions that are meaningless or broken under Cursor. Nothing else.

### 3. Additions

Ask these three questions of the source body. Add a paragraph only where the answer is no.

1. **Does it say what the skill acts on, and what to do when that is missing?** Both halves, not one. `generate-image` has both ("The subject is whatever the user gave you… if there is no description, ask for one and stop") — add nothing. `explain-code` names the input but never says what happens when it is absent — add the paragraph. Place it as the first paragraph after the `#` heading.
2. **Does it say when the skill is finished?** If not, add a stop condition as the last line of the body.
3. **Does it describe delegating to subagents?** If it never mentions delegation, add nothing at all. If it delegates but never says what happens when it is already inside a subagent, add that caveat — nesting stops after one level — immediately after the passage describing the delegation.

Do not add anything else. In particular, do not add "do not use subagents" to a skill whose source never mentions them.

### 4. The description

- Source is `disable-model-invocation: true` → carry the description over **verbatim**. The user triggers it by name; routing wording buys nothing.
- Source has it false or absent → rewrite it to name what the skill does **and** the situation that should trigger it, in the words a user would type. One or two sentences. Nothing else in the file changes.

## Step 4 — Verify

```bash
python3 -c '
import os,re,sys
dst=sys.argv[1]; fails=[]
def read(p):
    try: return open(p,encoding="utf-8").read()
    except Exception: return ""
for f in [os.path.join(dp,f) for dp,_,fs in os.walk(dst) for f in fs]:
    t=read(f)
    for tok in ("$ARGUMENTS","${CLAUDE_"):
        if tok in t: fails.append("leftover "+tok+" in "+os.path.relpath(f,dst))
t=read(os.path.join(dst,"SKILL.md"))
m=re.match(r"^---\n(.*?)\n---\n(.*)$",t,re.S)
if not m: fails.append("SKILL.md has no frontmatter block")
else:
    fm,body=m.group(1),m.group(2)
    allowed={"name","description","paths","disable-model-invocation","metadata"}
    for l in fm.split("\n"):
        if l[:1] not in ("", " ", "-", "#") and ":" in l:
            k=l.split(":",1)[0]
            if k not in allowed: fails.append("frontmatter field not in Cursor format: "+k)
    nm=[l.split(":",1)[1].strip() for l in fm.split("\n") if l.startswith("name:")]
    if not nm or nm[0]!=os.path.basename(dst.rstrip(os.sep)): fails.append("name does not match folder: "+(nm[0] if nm else "(missing)"))
    for ref in sorted(set(re.findall(r"(?:scripts|references|assets)/[A-Za-z0-9._/-]+",body))):
        if not os.path.exists(os.path.join(dst,ref)): fails.append("referenced file missing: "+ref)
print("OK" if not fails else "FAIL\n"+"\n".join("- "+x for x in fails))
' <ROOT>/cursor/skills/<name>
```

On `FAIL`, fix what it names and run it again. Never report a port that has not printed `OK`.

## Step 5 — Report

One block per skill:

```markdown
### <name> — CREATED
`cursor/skills/<name>/SKILL.md`

- Frontmatter dropped: `argument-hint`
- Description: carried over verbatim (source is `disable-model-invocation: true`)
- Substitutions: `$ARGUMENTS` ×1, `${CLAUDE_SKILL_DIR}` ×2
- Additions: stop condition
- Siblings copied: `scripts/openrouter-imagegen.py`, `scripts/requirements.txt`
- Verify: OK
```

Omit any bullet whose value is empty. `Verify:` always appears — its absence must never be mistakable for a pass. On an `UPDATED` block, add one line when it applies:

```
- Changed vs previous Cursor version: `disable-model-invocation` absent → true
```

Stop there. Do not invoke the ported skill to test it.
