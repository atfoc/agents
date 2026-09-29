---
name: implement-tasks
description: Implements every for-agent task by looping over the startable tasks, spawning one subagent per task in parallel, each code-changing task in its own git worktree, landing reviewed work on the base branch and turning failed reviews into fix tasks, until nothing is left. Use when tasks already exist and you want them implemented.
argument-hint: "[tasks location] [repository]"
---

# Implement tasks

This skill acts on the tasks at the location the user provides, `$ARGUMENTS`, or the location given earlier in the conversation. If no location is given, ask for it and stop. Never default or invent one.

The code goes into one git repository: the one the user names, or else the git repository of the working directory. If there is neither, ask for it and stop. The **base branch** is the branch checked out there when the run starts. Before creating the first worktree, check that checkout: if it has uncommitted changes to tracked files, report them and stop, since no worktree would see them.

The tasks to implement are tagged `for-agent`. A second tag says what kind of task it is: `contracts`, `implement`, `review`, `fix` or `merge`. A `for-agent` task with none of these is a plain task.

You run the loop, manage the tasks and land branches. Subagents do the work: one subagent per task, never two tasks in one, never one task across two. Every subagent inherits your model and effort: spawn it with no model and no agent type that sets its own.

## Hard rules

- Never reason about blocking yourself; the startable list decides what can run.
- Work only tasks tagged `for-agent`. Any other task is someone else's: never fetch it as work, spawn it or complete it. It still blocks.
- Run no verification yourself. A reported verification failure means the task is not completed.
- Never edit a task's body.
- The only writes you make: create a task and tag it, block a task, complete a task, create and remove worktrees, land branches (section 5), and code edits of at most two lines, committed in the worktree of the task they fix.
- Never change anything outside the repository, its worktrees and the task store: no shell config, PATH, installed tools, cluster or cloud state, or other repositories. A problem that needs one of those becomes a task without the `for-agent` tag.

## 1. Work out the body command

Before spawning anything, settle the one command that prints a single task's body and nothing else:

- Use the command that reads a task body by task id, with every value it requires filled in. Change nothing else.
- Write it literally: absolute paths, no variables, nothing relative to a working directory.
- If reading a task prints the whole task (id, title, blockers, status along with the body), write a small adapter script for this run in the scratchpad. It takes the task id as its one argument, reads the task and prints the body alone. The body command is then a call to that adapter. Do not clean it up and do not report its path.
- If no command a subagent could run reaches a body, report that the task bodies cannot be read this way, spawn nothing, and stop.

Per task, only the id in this command changes.

## 2. Where each task runs

- **`contracts` and `implement`** — a worktree of its own, on branch `task/<task id>` from the current base branch, at `<repository>-worktrees/<task id>` next to the repository. Create it just before spawning. If a worktree or branch of that name is left from an unfinished attempt, remove both first.
- **`review`** — the worktree of the `implement` task it is blocked by.
- **`fix` and `merge`** — the worktree named in its body.
- **Plain tasks** — the repository's own checkout, on the base branch.

## 3. Run the loop

1. **Fetch the startable tasks tagged `for-agent`.** That list is authoritative. If the startable list cannot be filtered by tag, fetch it whole and keep only entries tagged `for-agent`.
2. **Spawn one subagent per startable task not already in flight**, all in parallel in a single message, with no cap.
3. **Record the spawned ids in an in-flight ledger.** A running task is still startable; the ledger stops it being spawned twice.
4. **On a report**, handle it as section 6 says, drop that task from the ledger, land any branch still waiting if no plain task is in flight, fetch the startable list again, and immediately spawn anything newly startable without waiting for other in-flight work.

Only the initial fetch and a completion change what is startable.

## 4. The subagent prompt

The whole prompt is exactly this, with `<BODY COMMAND>` replaced by the body command carrying that task's id, and `<WHERE>` by the block below that matches where the task runs:

```
Read your task by running:

    <BODY COMMAND>

That is your whole assignment.

<WHERE>

- Carry it out fully, including its verification.
- Do no other task's work.
- Never create, block, tag or complete tasks.
- Never change shell config or PATH, and never install or upgrade tools.
- On a conflict, a gap, or anything the task does not cover, stop and report it rather than
  resolving it.
- Report back what was done, and every defect you noticed and did not fix.
```

`<WHERE>` for a `contracts` or `implement` task:

```
Work only in <WORKTREE>, on branch <BRANCH>. Where the task names <REPOSITORY>, use <WORKTREE>
instead. Before changing anything, run the build and the tests once; if they already fail, stop
and report that. Commit all your work to <BRANCH> before reporting.
```

`<WHERE>` for a `review`, `fix` or `merge` task:

```
Work only in <WORKTREE>, on branch <BRANCH>. Where the task names <REPOSITORY>, use <WORKTREE>
instead. Commit all your work to <BRANCH> before reporting.
```

`<WHERE>` for a plain task:

```
Work in <REPOSITORY>, on branch <BASE BRANCH>. Commit nothing.
```

Add nothing else. Never paste, summarise, quote or preview a body; never name the spec or feature definition; never mention other tasks; never explain why this task was picked.

## 5. Landing a branch

Work reaches the base branch only when you land it, one branch at a time. Never land while a plain task is in flight; the branch waits, and the task whose success is waiting on it stays uncompleted until it lands.

1. In the task's worktree, merge the base branch into its branch.
2. In the repository's checkout, fast-forward the base branch to that branch.
3. Remove the worktree and delete the branch.

If step 1 conflicts, abort the merge; never resolve it yourself. Create a `for-agent` task tagged `merge` whose body names the worktree and branch and says: merge the base branch into this branch, resolve the conflicts keeping what both sides intended, then make the full build and every test pass. Block by it the task whose success was waiting to land.

If step 2 is refused, report why and land nothing more.

## 6. Handle reports

- **`contracts` task succeeded** — land its branch, then complete it.
- **`implement` task succeeded** — complete it. Its branch lands when its review passes. If no review task is blocked by it, land it now, as for `contracts`.
- **`review` task passed** — land the branch of the task it reviewed, then complete the review.
- **`review` task reported failing tests** — create a `for-agent` task tagged `fix`. Its body names the worktree and branch, gives the failing tests exactly as the review reported them, gives the body command for the reviewed task so the fixer can read what was asked, and says: make these tests pass without changing them; if a test is wrong, stop and report why. Block the review by it. A review gets at most two fix tasks: when it fails after its second, create a task without the `for-agent` tag instead, holding the failing tests and both fix reports, and block the review by that.
- **`fix` task succeeded** — complete it. The review it blocks becomes startable and runs again.
- **`merge` task succeeded** — land its branch, then complete it and the task it was blocking.
- **Plain task succeeded** — complete it.
- **A contract has to change** — create a `for-agent` task tagged `contracts` whose body says which contract changes, how and why, and that it updates every existing implementation and caller so the full build and every test pass. Block by it the reporting task and every pending task not in flight whose body uses that contract.
- **A defect reported and not fixed** — create it as a pair, the way the breakdown does: an `implement` task stating the defect and the behaviour expected instead, and a `review` task blocked by it that checks that behaviour and writes tests for it. Both are `for-agent` when an agent can make the fix; otherwise create one task without that tag. Block by the review, or by that task, any pending task the defect affects.
- **A task failed**, reported a verification failure, or stopped on a problem — a one-or-two-line fix may be applied inline. Anything larger becomes a remediation task: a code change as a pair, as above; anything needing a person as a task without the `for-agent` tag. Block the failed task by the remediation, or by the pair's review, so it leaves the startable set instead of being respawned. If the remediation cannot be expressed as a task, exclude that task for the rest of the run and report it.
- One failure never stops the run: in-flight work continues and the loop keeps spawning whatever is startable.

## 7. Stop

- **Nothing pending tagged `for-agent`** — report one line per completed task, everything reported and not resolved, and every worktree still on disk.
- **Something pending tagged `for-agent`, nothing startable, nothing in flight** — report the stall, naming each pending task and what blocks it. Where a blocker is untagged, say it belongs to someone else and the run can be started again once they complete it.

The skill is finished when one of these stop conditions is reported; stop there.
