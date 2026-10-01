---
name: with-docs
description: Complete a task using the docs collections — resolve every convention, format, procedure and standard the task needs out of the docs before doing the work, and report anything the docs could not answer. Use when a task depends on documented knowledge rather than on the code in front of you.
---

# Complete a task with the docs

Invoke with `$with-docs <the task>`. The task is whatever the user gave you in the current request, or the request earlier in the conversation.
If there is no task, ask what to do and stop.

Docs are a source of information, not a procedure. Search them until the task is fully specified,
then do the task. Nothing here says which doc to open — that is what the indexes are for.

## Where the docs are

- **User docs** — the `AI_DOCS` environment variable: a `:`-separated string of paths. Every path
  in it is a docs root.
- **Project docs** — under the working directory, every folder holding a `DOCS.md`:

      find . -name DOCS.md -not -path '*/.git/*' 2>/dev/null

  The shallowest match on a branch is a root. Deeper ones are reached through their parent's index.

**A directory with no `DOCS.md` holds no docs.** It is a helper for its parent — scripts, templates,
assets. Never search it for information; use its files only when a doc points you at them.

## How to search

1. Read each root's `DOCS.md` first. It is the index: what that directory holds and where. It
   routes; it does not carry the content. Entries may add a `Use when ...` trigger naming the
   situation they are for — match it against yours.
2. Open only the entries that answer a question you actually have. Do not read a directory
   exhaustively. Descend into a subdirectory only through its own `DOCS.md`.
3. **Fallback** — for what the indexes do not answer, text-search the roots:

       grep -ril "<term>" <root>

   Read the hits that look relevant, subject to the same helper-directory rule.

## Search in a loop

One pass is rarely enough. After each read, ask what it introduced that you still cannot act on,
and search for that. Repeat until nothing is left unresolved.

Search again when:

- **A doc references another doc.** Read it.
- **A doc names a concept it does not define.** Search the docs for that concept before continuing.
  A doc that says "create a local task" without saying how has just raised a new question — what
  local tasks are, and what command creates one. Search for it; do not assume.
- **An answer is partial or conditional.** Resolve the condition.

Stop only when you could carry out the task with no remaining guesses.

## Conflicting information

When sources disagree, the higher one wins:

1. Facts stated in the prompt
2. Facts stated in project docs
3. Facts stated in `AI_DOCS`

## What the docs could not answer

If a question is still open after the loop, list those questions, ask the user to answer them, and
stop until they do.

Never guess a convention the docs were supposed to provide, never silently substitute your own
default, and never report a task as done on information you invented.

## Do the task

Complete it using what you resolved. Where a doc decided something, say which doc — by path — so
the user can check it.
