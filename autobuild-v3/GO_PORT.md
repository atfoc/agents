# What to change in the Go port (AI Whiteboard runs) to match autobuild v3

The Go port in the AI Whiteboard app (`internal/runs`, `internal/model/run.go`, `web/src/run`) was
written from autobuild v2. This file lists what v3 changed in the script and where the same change
goes in the port. Nothing in the port has been changed yet.

File and line references to the port are to commit `00df6e6125d4` of the branch
`autobuild2/runs-feature/integration`, read while that run was still in progress: check them
against the code before starting. References to the script are to `autobuild-v3/autobuild.py`.

## Why

Measured on the v2 run that built the port (62 finished agents, $324 at list price, all on Opus 5.5
at high effort):

- Output tokens were 35% of the cost, cache writes 36%, cache reads 29%.
- The median writing task had 207K tokens of context before its first file write and 403K at the
  end; 79% of cache-read cost came from requests made with more than 200K of context.
- 13 of 30 orchestrator turns changed nothing in the plan ($11 of $46).
- The orchestrator rewrote its notes whole (46K characters by the end) on most turns.
- At most 6 tasks were ever ready at once; the cap of 4 delayed tasks on the critical path.

## 1. Tiers: a model and an effort per task

Script: `TIERS`, `DEFAULTS["tiers"]` (line 61), `run_agent` (589), `check_definition` (1461),
`op_add_task` (1642), `op_retry_task` (1737), the tier list in `p_orchestrator` (955),
`merge_config` (2125).

What v3 does:

- Three tiers, `deep`, `standard`, `light`. The run's configuration maps each to a model and an
  effort; the orchestrator picks a tier and never names a model.
- `add_task` requires `tier` and `tier_reason`. `update_task` accepts both. `retry_task` accepts
  `tier`, and the new attempt then runs on the new tier.
- The orchestrator runs at `deep`, merge agents at `standard`.
- An agent keeps the model it was first started with: a resumed session is never moved to another
  model. A tier change takes effect only for a new attempt, which is a new agent.
- The orchestrator's prompt lists the tiers with their model and effort, and three rules: give the
  lowest tier that is safe and say why, take the higher when unsure, go one tier up after a failure
  or a review that finds real faults.
- The run snapshot shows each task's tier and cost.

In the port:

- `internal/model/run.go`: `RunMeta` holds one `Agent`, `Model` and `Effort` for the whole run
  (line 134). Replace `Model` and `Effort` with a tier map; keep one `Agent` kind per run. `RunView`
  and `RunDefaults` follow.
- `Task` in `internal/runs/types.go` gets `Tier` and `TierReason`; each `Attempt` records the tier
  it ran on. The wire types and `web/src/run/RunTask.tsx` show them.
- `internal/runs/agentrun.go:89`: `CreateOwned` is given `meta.Model` and `meta.Effort`. Take them
  from the tier of the task (or the fixed tier of the role) instead.
- `internal/runs/tools.go` (`addTask`, `updateTask`, `retryTask`), `tooltext.go`, `prompts.go`:
  the new fields, their refusals and the prompt text. The golden tests
  (`tools_golden_test.go`, `prompts_test.go`) change with them.
- `web/src/run/RunComposer.tsx`: one model picker and one effort picker become three rows, one per
  tier.

To decide in the port:

- The port runs Claude Code, Cursor and pi agents. The tier map needs defaults per agent kind,
  taken from each kind's model catalogue, and a kind with a single usable model maps all three
  tiers to it.
- Effort exists only where the model has effort levels (`m.efforts` in the composer).

## 2. Wake: the orchestrator says what it waits for

Script: `op_wait_for` (1762), `wait_met` (1294), `next_turn` (1302), `tools` (1420), the `wake` and
`ending` texts in `p_orchestrator` (955).

What v3 does:

- A third wake mode, `declared`, is the default. `each` and `idle` remain.
- New tool `wait_for {tasks, mode: all | any}`, offered only in `declared` mode. It refuses unknown
  ids and tasks that have already ended. The last call of a turn wins.
- A turn starts when the wait is met (done, failed and cancelled all count as ended), when any task
  fails, or when nothing is running. A turn that declared no wait is followed by a turn after every
  finished task, as in `each`.
- The wait is part of the recorded state, survives a restart, and is cleared when the next turn
  starts, whatever started it. The turn records the wait it was started under, and its prompt says
  whether the wait was met or the turn started early and why.

In the port:

- `internal/model/run.go:108` and `DefaultRunSettings` (line 116): add `declared` and make it the
  default.
- `internal/runs/turn.go:100` (`turnReason`): add the reason `wait`, checked before `idle`. Keep
  what the port added: a `chat_op` in the inbox and a person's resume start a turn in every mode.
- The wait goes into the journal state next to `Inbox`, and into the turn's record.
- `tools.go`: `wait_for` is the orchestrator's alone, like `set_notes`; a person's chat is refused.
- `web/src/run/Timeline.tsx` and `RunTurn.tsx`: show what a turn waited for and what started it.

## 3. Notes: edit a section

Script: `replace_section` (395), `op_edit_notes` (1626), `save_notes` (1633), `NOTES_LIMIT` (89).

What v3 does:

- New tool `edit_notes {heading, text}` replaces the section under that heading, up to the next
  heading of the same or a higher level. A missing heading is added at the end as a `##` section,
  empty text removes the section, two sections with the same heading are refused, and headings
  inside code fences are ignored.
- `set_notes` stays for the first version and for reorganising.
- Both answer with the size of the notes, and above 20,000 characters tell the orchestrator to
  shorten them.

In the port:

- `tools.go:502` (`setNotes`) and `prompts.go:231`: add `edit_notes` as the orchestrator's tool.
  Each edit is a new notes version, as a `set_notes` is.

## 4. Parallel tasks

- `DefaultRunSettings` (`internal/model/run.go:116`): `MaxParallel` from 4 to 8. The composer's
  default follows.

## 5. Narrow tasks

Script: `dependency_results` (916), `reports_warning` (942), `needs_report` in `check_definition`
(1461), the "Keep a task narrow" paragraph in `p_orchestrator`, the last rule of `p_task` (1055).

What v3 does:

- `depends_on` orders tasks and gives the agent each dependency's summary and the path of its
  report. The new `needs_report`, a subset of `depends_on`, names the reports put in front of the
  agent whole (60,000 characters inline in total, by path beyond that).
- `add_task` and `update_task` warn when a task is given more than three reports or more than
  60,000 characters of them.
- The orchestrator is told to keep a task to one area, to name its files, and to quote in the brief
  what the task needs from earlier reports.
- A task agent is told that everything it reads is paid for on every later step, and to hand a part
  to a subagent only when the part is independent.
- The result schema says that a dependent task may be given only the summary.

In the port:

- `internal/runs/attempt.go:501`: every dependency's report is read into the prompt. Read it only
  for the ids in the task's `NeedsReport`.
- `Task` gets `NeedsReport`; `tools.go` validates it against `DependsOn` and drops ids that an
  update removes from `DependsOn`.
- `prompts.go` and `tooltext.go`: the texts and the warning.

## 6. Usage figures

Script: `tokens_of` (429), `context_of` (442), `_exec` (670), `usage_lines` (1930), `PLAN_OPS` (221).

What v3 does:

- Each agent record holds input, output, cache-read and cache-write tokens and the peak context of
  its own requests. Token counts come from `modelUsage` of the result event, which includes the
  agent's subagents; `usage` does not.
- `status` and the end of a run print cost by tier and by kind, median peak context, and how many
  turns left the plan as it was.

In the port:

- The agent adapters report cost today. Token counts and peak context are available from Claude
  Code's stream; for Cursor and pi record what each adapter can report and leave the rest empty.
- Show the same figures in the run view.

## Not ported from v3

- `.autobuild3` and the `autobuild3/` branch prefix are the script's own names.
- The script's command-line flags (`--tier`, `--model`, `--effort`) correspond to the composer.

## Order

1. Parallel default (4), usage figures (6).
2. Notes (3) and wake (2): each is one tool and one text change.
3. Narrow tasks (5).
4. Tiers (1): the largest, since it changes the run's model in `RunMeta`, the composer and the
   wire types.
