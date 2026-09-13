# Dev process state

Everything about one goal is bounded to `./.tasks/{goalName}/`. `{goalName}` is a short kebab-case
name for the goal; every session working the goal is given it.

## The goal folder

- `dev-process.md` — the state file, below.
- `task-outputs/{taskKey}/…` — where the task with that key writes its output. `{taskKey}` is a
  short kebab-case key chosen for the task before it is created, so the path can be written into
  the task's body literally; the state file maps it to the store's id.
- the task store, when it is of a kind that lives in files — a folder in the goal folder.
- a slice store per implementation flow, when it is of a kind that lives in files — a folder under
  the output folder of the task that split it.

## The state file

`dev-process.md` holds, and nothing else does:

- the goal, as the user gave it;
- the store description, verbatim;
- every task created: its key, its store id, its kind, its title;
- a log: one short entry per finished task saying what it concluded.

Rules:

- Rewritten in full at the end of every planning iteration.
- Read first by every later session on the goal, whichever doc runs it.
- Nothing about the goal is kept only in the conversation.
- The store description is copied from it verbatim, never reworded, into every session and every
  task body that needs it.

## The store description

The user's own words on where the goal's tasks live and which doc or skill describes that format.
It is complete only when it also says how a **separate** store for one implementation flow is
placed, worded so that it can be filled in per flow:

- for a store that lives in files — a folder under the output folder of the task that split the
  flow, `./.tasks/{goalName}/task-outputs/{taskKey}/…`;
- for a store that lives elsewhere — whatever the user said; for example, a task under which
  every slice becomes a subtask.

A description missing either part is incomplete. A store is never defaulted or invented.
