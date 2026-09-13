# Dev process orchestrator

Drives a goal — what the user wants built — to shipped code without the user deciding at every
step what comes next. It turns the road to implementation into tasks: research, feature and
implementation definitions, prototypes, splitting into slices, implementing them. Agents run the
tasks that need no person (`running-agent-tasks.md`); the user runs the rest
(`running-human-tasks.md`); the orchestrator re-plans on what came back.

One session is one iteration: read what finished, decide what comes next, create those tasks,
report. Everything between iterations lives in the state file — `dev-process-state.md`.

## Inputs

- **The goal** — what the user wants built.
- **The store description** — in the user's own words, as `dev-process-state.md` defines it,
  including how a separate slice store is placed per implementation flow.

In the first iteration, if the goal, the store description, or the slice-store placement inside it
is missing, ask for that one thing and stop. Never guess a default store and never start from an
empty goal. This is the only doc that takes the store description from the user directly; every
later iteration and every runner reads it from the state file.

## Goal name

In the first iteration, infer a short kebab-case `{goalName}` from the goal and state it in the
report. A name the user gives wins. Every later orchestrator and runner session is given that
name.

## One iteration

1. Read `./.tasks/{goalName}/dev-process.md`. First iteration: create it from the inputs.
2. Check the store, using the store description verbatim. If any task created so far is not
   completed, create nothing: report which tasks are pending, split by owner — `for-agent` ones,
   to be run by `running-agent-tasks.md`, and human ones, waiting on the user — and stop. Planning
   happens only on a store with nothing pending.
3. Read the outputs of the tasks that finished since the last iteration, from
   `./.tasks/{goalName}/task-outputs/{taskKey}/…`.
4. Decide the next tasks on your own. The kinds are fixed — `task-kinds.md` — so the decision is
   which kinds, with which titles, and which task blocks which. The blocker graph is the order;
   whatever it does not order runs in parallel. Do not consult the user: these are not
   implementation tasks, they are the steps that make implementation possible.
5. Choose a key for every task and create the tasks in blocker order, so that a blocked task's
   blockers already have ids when it is created.
6. Rewrite the state file in full and report.

## Task bodies

A task body is the whole assignment; whoever runs it reads nothing else. It carries:

- the kind, and the doc that kind runs, named as `task-kinds.md` names it;
- the goal context the task needs — no more;
- its inputs as literal paths: the output locations of its blockers,
  `./.tasks/{goalName}/task-outputs/{blockerKey}/…`, which exist by the time the task is
  startable;
- its own output location, `./.tasks/{goalName}/task-outputs/{taskKey}/…`, stated as overriding
  any default output path the doc behind the kind would otherwise use;
- for the kinds that create or work a store — split tasks, implement tasks — the store
  description verbatim, with the slice-store placement filled in for that flow.

`for-agent` tasks are tagged so on creation; human tasks carry no tag.

## Two stores

The orchestrator's own tasks and the implementation slices never share a store. The implementing
loop fetches every startable `for-agent` task from the store it is given; a research task in the
same store would be picked up as a slice to implement. Each `split tasks` task fills a separate
slice store, and the matching `implement tasks` task runs the implementing loop on that store
only. The implementing loop is never run against the orchestrator's store.

## Outputs that report a problem

A finished task whose output reports a problem — an unanswered question, a split that hit a
conflict, an implementation run that stalled — is handled inside the six kinds:

- a gap an agent can close becomes a `research` task;
- a gap that needs a person becomes the interactive kind that owns it, with the problem in its
  body, blocking the retry;
- a stalled implementation flow gets a new `implement tasks` task, blocked by whatever unblocks
  it.

A problem that fits none of the six kinds is reported as the iteration's result, and the
orchestrator stops; the user decides.

## Report

- One line per task created this iteration: id, kind, title, `for-agent` or human, what blocks it.
- Which of them are startable now.
- After the first iteration: one line per task finished since last time, with what it concluded.
- What to run next: `running-agent-tasks.md` for the `for-agent` tasks, and which human tasks wait
  on the user.

## Done

Several implementation flows may run and finish across iterations; a finished flow does not end
the goal, since more may still need defining. The goal is done when:

- every task in the store is completed;
- every `implement tasks` task planned has finished;
- reading the outputs raises no question that would need another task.

The final report says so, lists the slice stores that were implemented, and lists the prototype
branches and worktrees left behind so they can be cleaned up.

When the orchestrator cannot decide — an output is missing or contradicts the goal — it does not
invent a task to cover it. It reports the problem and stops.
