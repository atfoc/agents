---
name: spec-to-tasks
description: Used when we need to split an implementation spec into tasks. You say where the tasks go and which skill or doc describes that storage
argument-hint: [path-to-implementation-spec]
disable-model-invocation: true
---

# Split an implementation spec into tasks

You orchestrate two subagents and nothing else. You do not cut the spec, and you do not know how
or where tasks are stored — one subagent does the cutting, the other owns the storage. You never
read the implementation spec, the feature definition or the split file.

## Step 1 — Collect your two inputs

You need both of these before anything else happens.

**The implementation spec.** A path — the argument given to this skill, or a path named in the
conversation. Nothing else counts: never guess a filename, never search the repo for something
spec-shaped.

**Where the tasks go.** The user's own statement of where tasks live and what describes that
format — a folder plus "use the `local-task-format` skill", a Linear project plus the skill or doc
that covers Linear, anything of that shape. Take it from the invocation or from the conversation.

Check only that each input is *present*. Never check that the storage input is correct, complete
or usable, and never resolve, expand, rewrite or tidy it — not the paths inside it, not its
wording. Record it exactly as the user wrote it; you will hand that text on unchanged.

If either input is missing, ask for the missing one — both, if both are missing — and stop. Never
default the task location to the spec's folder or anywhere else, and never invent a store.

## Step 2 — Pick the split file

Create a fresh empty file in the session's temp area:

    SPLIT_FILE="$(mktemp "${TMPDIR:-/tmp}/spec-split-XXXXXX.md")"

If this session was given a scratchpad directory, create it there instead, with the same
`spec-split-XXXXXX.md` shape.

Rules for this file:

- Never ask the user where it should go, and never accept a location for it.
- One fresh file per run. Never reuse a previous run's file and never append to one.
- Never delete it, and never mention its path in anything you say to the user. A retry has it in
  context already.
- You never read it, and never quote from it. Only the two subagents read and write it.

## Step 3 — Spawn the splitter

Spawn one plain general-purpose subagent, alone: nothing else is spawned in that message, and
nothing else happens until it reports.

Pass a feature-definition path when either a path was named in the conversation or a
`feature-design.md` sits beside the spec. Paths only — never contents. Do not check that any path
exists; a bad path is the splitter's own hard failure.

The prompt is the template below with four substitutions and nothing else changed:

- `<SPEC PATH>` — the implementation spec path;
- `<FEATURE DEFINITION PATH>` — the feature definition path. When there is none, **delete that
  whole sentence** from the prompt rather than leaving a placeholder or the word "none";
- `<SPLIT FILE PATH>` — the file from Step 2;
- `<SPLIT FILE FORMAT>` — the split-file format block, pasted in whole.

```
Your job is to cut an implementation spec into slices and write them to one file. You do not
create tasks, you do not know or ask where tasks are stored, and you never look for a task store.
The file you write is your only output.

Read `<SPEC PATH>` end to end. Also read `<FEATURE DEFINITION PATH>` end to end. Do not begin
cutting before you have read both completely.

## Cut into vertical slices

A slice is a collection of work that builds one functional and testable unit of the whole
implementation spec.

- A slice that needs another slice's work to be functional cuts that branch with a stub, and the
  body names the stub explicitly so the slice that later replaces it knows what it is replacing.
- Every slice carries its own verification. A slice is done when its verification passes, so there
  are no verification-only slices.
- Work that cannot be verified alone folds into the slice whose verification exercises it.
- **Never re-cut a slice to gain parallelism.** Keeping the change → verify → fix loop inside one
  implementer's context beats handing pieces around. Parallelism is only what falls out of slices
  that were already independent.

## Write each body

- Carry text from the implementation spec verbatim — no summarising, no rewording, no "see the
  spec".
- The union with deduplication of all bodies reproduces the whole implementation spec.
- Duplication across slices is expected and correct: a section that applies to four slices is
  copied into all four.
- Feature-definition text is carried verbatim wherever a slice needs its reason or its
  user-visible behaviour.
- You may add your own detail, and added detail only narrows or sequences what the documents
  already decided.
- You may never add anything that contradicts, weakens or reinterprets the implementation spec or
  the feature definition. If a workable slice would require that, stop, write nothing, and report
  what conflicts with what.
- Never name the implementation spec's or the feature definition's path in a body. Whoever
  implements a slice gets that body and nothing else.

## Decide blockers

A slice's blockers are the slices that must be completed before it may start, for exactly two
reasons:

1. It consumes an artifact another slice produces — a type, a table, a stub it replaces, a module
   it imports.
2. The two would edit the same file and so cannot run at the same time. Same file, one blocks the
   other; when the order is arbitrary, pick one and say nothing more about it.

The graph must be acyclic. A cycle means a bad cut, so merge or re-cut; a cycle that cannot be
resolved is a stop-and-report. At least one slice must have no blockers.

## Check before you write

All four, before you write anything:

- Every section of the implementation spec appears in at least one body. An orphaned section is a
  stop, not a silent drop.
- No cycles.
- At least one slice with no blockers.
- Nothing contradicted.

Any failure → write nothing and report why.

## Write the file

Write all slices to `<SPLIT FILE PATH>`, replacing whatever that file holds, in exactly this
format:

<SPLIT FILE FORMAT>

## Report back

Report that you wrote the file, how many slices it holds, every piece of detail you added beyond
the source documents, and anything you had to stop on. Never report the file's contents back —
they are in the file.
```

## The split-file format

This is the `<SPLIT FILE FORMAT>` block. It is pasted whole into both subagent prompts.

```
The file is markdown. It holds nothing but slice blocks, one after another, and no preamble.

    ===== SLICE <ref> =====
    title: <one line>
    blocked-by: <comma-separated refs, or the word none>
    ----- BODY -----
    <the full body, verbatim, as many lines as it takes>
    ===== END SLICE <ref> =====

- `<ref>` is a local reference — `1`, `2`, `3`, … — that exists only inside this file and means
  nothing outside it. Number them sequentially from 1, each used exactly once.
- `blocked-by` lists local references only, never titles: `blocked-by: 1, 3`, or `blocked-by: none`.
- Blocks are written in dependency order: every slice appears after all the slices it is blocked
  by.
- The body is everything between the `----- BODY -----` line and the matching
  `===== END SLICE <ref> =====` line, copied exactly. It may contain headings, `---` rules and
  fenced code blocks; the five-equals and five-dash sentinel lines are what separate blocks, so
  nothing inside a body is ever treated as structure.
- A block carries these four things and nothing else. No status, no ids, no notes, no metadata.
```

## Step 4 — Stop if the splitter did not succeed

If the splitter reports a conflict, a stop, or any failure, report exactly what it reported and
stop. If it reports success but the split file does not exist or is empty (`test -s`), report that
and stop.

Never spawn the creator after any of those. Never try to resolve the problem yourself, never edit
the split file, and never retry the splitter with different inputs, a different spec or a
different cut.

## Step 5 — Spawn the creator

Only once Step 4 has passed. One plain general-purpose subagent, alone.

The prompt is the template below with three substitutions and nothing else changed:

- `<SPLIT FILE PATH>` — the file from Step 2;
- `<SPLIT FILE FORMAT>` — the format block above, pasted in whole;
- `<STORAGE INPUT>` — the user's text from Step 1, pasted character for character, every sentence
  of it if they said it across several turns.

On `<STORAGE INPUT>` specifically: do not resolve a skill name into its content, do not expand or
absolutise a path, do not turn "the tasks folder next to the spec" into an actual path, do not add
a clarifying gloss, do not drop the part you think is redundant.

You never load the storage skill or doc yourself — not to check it exists, not to read its
operations. Only the creator resolves it. A name that turns out to be wrong is the creator's
failure to report.

```
Your job is to transcribe already-cut tasks into a task store. You did not cut them, and you never
change them.

## Where the tasks go

In the user's own words:

<STORAGE INPUT>

Resolve that yourself: load whatever skill or doc it names and follow it. Everything you do to the
store goes through that format's own operations — never your own file reads or writes, never a
store of your own design, never a guessed format.

If that input is not enough to create tasks — it names no format, the skill or doc it names does
not exist, it gives no location, or the format needs an input it does not supply — stop, create
nothing, and report exactly what is missing.

## What to transcribe

Read `<SPLIT FILE PATH>`. It is in this format:

<SPLIT FILE FORMAT>

## Before you create anything

Using the format's own operations, ask whether the store exists and whether it is empty.

- Absent → proceed.
- Exists and empty → proceed.
- Exists and holds tasks → create nothing and report which store it is, that it already holds
  tasks, and that re-splitting requires emptying it first.

## Create

Walk the blocks in file order, which is already dependency order, and create one task per block:

- the title exactly as the block's `title` line reads;
- the body exactly as it appears between the body sentinel lines, every character of it;
- blockers: the block's local references mapped through to the real ids the store handed back for
  those blocks. `none` means no blockers.

Keep the map from local reference to real id as you go; you need it for every later block and for
your report.

You are a transcriber. Never reword, trim, expand, summarise, merge, split, reorder, add or drop a
task. Never fix a typo, never tidy formatting, never add a heading, a prefix or a note about where
the task came from. A body that looks wrong to you is still created exactly as written.

If a create fails, stop there. Report which tasks were created with their ids, which block failed
and the exact error. Never undo what you created, never retry with altered content, never skip the
block and continue.

## Report back

Report where the tasks were created — describing the store the way the input above describes it —
how many you created, one line per task with its real id, its title and its real blocker ids, and
which tasks are startable now, meaning the ones you created with no blockers.
```

## Step 6 — Report and stop

Merge the two reports into one report to the user:

- where the tasks were created, in the creator's own description of it, and how many;
- one line per task with its real id, its title and its real blocker ids;
- which tasks are startable now;
- every piece of detail the split added beyond the source documents, carried through from the
  splitter's report so the user can check it.

Say nothing about which subagent did what and never name, quote or hint at the split file. Carry
the added-detail list through as the splitter gave it — do not shorten or re-summarise it.

Then stop. Never begin implementing and never invoke an execution skill, even when the user asked
for both at once.
