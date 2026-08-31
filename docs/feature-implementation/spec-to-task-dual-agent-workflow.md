# Spec to tasks with two agents

Going from an implementation spec to a store of tasks with two agents, one after the other: the
first cuts the spec into vertical slices, the second creates one task per slice. Neither does the
other's job, and whoever runs them does neither.

**The task store is whatever the user asked for** — the location and the format or doc that
describes it, in the user's own words. Never default it to the spec's folder, never carry over the
store a previous run used, never invent one. If the user did not say where tasks go, ask and stop
before anything is spawned.

## The cutting agent

Runs alone; nothing else happens until it reports.

- Gets the implementation spec path, the feature definition path when there is one, and the path
  of the handoff file. Paths only, never contents.
- Cuts and writes the bodies by `split-spec-into-vertical-slices.md`.
- Is told nothing about the task store, and never looks for one.
- Writes every slice to the handoff file. That file is its only output; the report says how many
  slices, what detail was added beyond the source documents, and anything it stopped on.

If it reports a conflict, a stop or any failure, or the handoff file is missing or empty, report
that and stop. Never spawn the second agent after it, never repair the handoff file, and never
re-run the cut with different inputs.

## The handoff file

A fresh file per run, in a temp or scratch area, never a path the user chose and never appended to.
Only the two agents read or write it; whoever runs them never reads it and never quotes it.

It is markdown holding nothing but slice blocks, no preamble:

    ===== SLICE <ref> =====
    title: <one line>
    blocked-by: <comma-separated refs, or the word none>
    ----- BODY -----
    <the full body, verbatim, as many lines as it takes>
    ===== END SLICE <ref> =====

- `<ref>` is a local reference — `1`, `2`, `3`, … — numbered from 1, each used once, meaning
  nothing outside this file.
- `blocked-by` lists local references only, never titles.
- Blocks are in dependency order: a slice appears after every slice it is blocked by.
- The body is everything between the sentinel lines, copied exactly. The five-equals and five-dash
  lines are the only structure, so headings, rules and fenced blocks inside a body are just body.
- A block carries those four things and nothing else — no status, no ids, no notes.

## The creating agent

Runs only once the cut succeeded, alone.

- Gets the handoff file path, the format block above, and the user's statement of where the tasks
  go, pasted character for character. Never resolve a named format into its content, never expand
  a path, never gloss it: the creating agent resolves it and follows it.
- Every read and write of the store goes through that format's own operations — never hand-written
  files, never a guessed format. If the input names no format, names one that does not exist,
  gives no location, or omits something that format requires, create nothing and report what is
  missing.
- Before creating, ask the store whether it exists and whether it is empty. Absent or empty →
  proceed. Holds tasks → create nothing and report that re-splitting requires emptying it first.
- Walk the blocks in file order and create one task per block: the title as written, the body every
  character as written, and blockers mapped from local references to the ids the store handed back.
- It is a transcriber. Never reword, trim, merge, split, reorder, add or drop a task, never fix a
  typo, never note where a task came from. A body that looks wrong is created exactly as written.
- A failed create stops the run: report the tasks created with their ids, the block that failed and
  the exact error. Never undo, never retry with altered content, never skip and continue.

## Report

One report, merged: where the tasks were created in the creating agent's own description of it, how
many, one line per task with its id, title and blocker ids, which are startable now, and the detail
the cut added beyond the source documents, carried through unshortened.

Nothing about which agent did what, and never name or quote the handoff file. Creating tasks is
where this stops — implementing them is `implementing-tasks.md`, started separately.
