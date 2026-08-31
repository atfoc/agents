---
name: with-docs
description: Complete a task using the docs collections — resolve every convention, format, procedure and standard the task needs out of the docs before doing the work, and report anything the docs could not answer. Use when a task depends on documented knowledge rather than on the code in front of you.
argument-hint: [the task]
disable-model-invocation: true
---

# Complete a task with the docs

The task is whatever the user gave you: `$ARGUMENTS`, or the request earlier in the conversation.
If there is no task, ask what to do and stop.

Docs are a source of information, not a procedure bound to this skill. You search them, build the
context this task needs, and then do the task. Nothing here tells you which doc to open — that is
what the indexes are for.

## Step 1 — Name what is missing

Before reading anything, write down the specific questions this task cannot be completed without:
the formats, conventions, procedures, standards, exact commands it depends on.

Each question must be specific enough that you would recognise its answer when you saw it. "How do
tasks work" is not a question. "What command creates a task in this store, and what does it print"
is.

If the task needs nothing you do not already have, say so and go to Step 6.

## Step 2 — Find the docs roots

- **User docs** — the `AI_DOCS` environment variable: a `:`-separated string of paths. Every path
  in it is a docs root.
- **Project docs** — under the working directory, every folder holding a `Docs.md`:

      find . -name Docs.md -not -path '*/.git/*' 2>/dev/null

  The shallowest match on a branch is a root. Deeper ones are reached through their parent's index,
  not directly.

If there are no roots at all, say so and go to Step 5.

## Step 3 — Navigate by index

Read each root's `Docs.md` first. It is the index: what that directory holds and where. It routes;
it does not carry the content.

- Open only the entries that answer one of your questions. Do not read a directory's docs
  exhaustively.
- Descend into a subdirectory only through its own `Docs.md`.
- **A directory with no `Docs.md` holds no docs.** It is a helper for its parent — scripts,
  templates, assets. Never search it for information and never descend into it looking for docs.
  Run or use its files only when a doc points you at them.
- Follow links between docs. A doc that says "see `x.md`" is naming something you need.
- Stop as soon as every question is answered.

## Step 4 — Fall back to text search

Only for the questions the indexes did not answer. Search the roots for the terms in those
questions:

    grep -ril "<term>" <root>

Read the hits that look relevant, subject to the same rule: a file inside a directory with no
`Docs.md` is a helper, not a doc.

## Step 5 — Report what is still missing

If any question is still open, list those questions and ask the user to supply the answers, then
stop until they do.

Never guess a convention the docs were supposed to provide, never silently substitute your own
default, and never report a task as done on information you invented.

## Step 6 — Do the task

Complete it using what you resolved. Where a doc decided something, say which doc — by path — so
the user can check it.
