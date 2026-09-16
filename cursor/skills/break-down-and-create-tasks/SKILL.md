---
name: break-down-and-create-tasks
description: Breaks a piece of work down into tasks in one subagent, then creates those tasks, each tagged `for-agent`, wherever the user said in a second subagent. Use when you want to turn a spec or body of work into tasks and create them wherever you keep tasks for agents to pick up.
---

# Break down work and create tasks

The input is the work the user gave with the request (a spec, a file path, a document or text earlier in the conversation) and the place the user wants the tasks created. If the work is missing, ask for it and stop. If the destination is missing, ask where the tasks should be created and stop.

Every task created by this skill gets the tag `for-agent`. No exceptions.

Never reword, summarise or paste the work or the tasks into a subagent prompt. Pass each subagent only a way to fetch what it needs.

## Step 1 — Break the work down in a subagent

Work out how the work can be fetched: a file path, a URL, or a document or task identifier together with where it lives. If the work exists only as text in the conversation, write that text unchanged to a file and use that file's path.

Spawn one subagent and wait for it to finish. Give it this prompt, with only the way to fetch the work filled in:

```
Fetch the work from <way to fetch the work> and break it down into individual tasks as vertical slices, each with its own title, self-contained body, verification and blockers. Do not create the tasks anywhere. Write the tasks to a handoff file, choosing its location and structure yourself, so that another agent can create every task from that file alone, with its title, body and blockers exactly as you wrote them. Return only the path of the handoff file.
```

Take the handoff file path it returns. If there is no file, or the file holds no tasks, tell the user and stop.

Subagent nesting stops after one level. If you are already running inside a subagent and cannot spawn another, do the work of this step and of Step 2 yourself, following the same prompts.

## Step 2 — Create the tasks in a subagent

Spawn a second subagent and wait for it to finish. Give it this prompt, with the destination and the handoff file path filled in:

```
Read the handoff file at <handoff file path> and create every task it describes in <destination>. Use each task's title and body exactly as written in the file, without rewording, and tag every task `for-agent`. After all tasks exist, mark each task as blocked by the tasks the file lists as its blockers. Report back every created task with its title, identifier or link, tags and blockers.
```

## Step 3 — Report

Show the user the created tasks from the second subagent's report. If any task is missing the `for-agent` tag or failed to be created, say so plainly.

The skill is finished once the report is shown; stop there.
