---
name: break-down-and-create-tasks
description: Breaks a piece of work down into tasks in one subagent, then creates those tasks, tagging `for-agent` the ones the breakdown marked for agents, wherever the user said in a second subagent. Use when you want to turn a spec or body of work into tasks and create them wherever you keep tasks for agents to pick up.
argument-hint: [work to break down] [where to create the tasks]
---

# Break down work and create tasks

The input is the work the user gave with the request (a spec, a file path, a document or text earlier in the conversation) and the place the user wants the tasks created. If the work is missing, ask for it and stop. If the destination is missing, ask where the tasks should be created and stop.

A task gets the tag `for-agent` only when the breakdown marked it `for-agent`. Every other task is created without that tag.

Never reword, summarise or paste the work or the tasks into a subagent prompt. Pass each subagent only a way to fetch what it needs.

## Step 1 — Break the work down in a subagent

Work out how the work can be fetched: a file path, a URL, or a document or task identifier together with where it lives. If the work exists only as text in the conversation, write that text unchanged to a file and use that file's path.

Spawn one subagent with the Agent tool and wait for it to finish. Give it this prompt, with only the way to fetch the work filled in:

```
Fetch the work from <way to fetch the work> and break it down into individual tasks as vertical slices, each with its own title, self-contained body, verification, blockers and whether it is `for-agent`. Do not create the tasks anywhere. Write the tasks to a handoff file, choosing its location and structure yourself, so that another agent can create every task from that file alone, with its title, body, blockers and whether it is `for-agent` exactly as you wrote them. Return only the path of the handoff file.
```

Take the handoff file path it returns. If there is no file, or the file holds no tasks, tell the user and stop.

## Step 2 — Create the tasks in a subagent

Spawn a second subagent with the Agent tool and wait for it to finish. Give it this prompt, with the destination and the handoff file path filled in:

```
Read the handoff file at <handoff file path> and create every task it describes in <destination>. Use each task's title and body exactly as written in the file, without rewording. Tag a task `for-agent` only if the file marks it `for-agent`; create every other task without that tag. After all tasks exist, mark each task as blocked by the tasks the file lists as its blockers. Report back every created task with its title, identifier or link, tags and blockers.
```

## Step 3 — Report

Show the user the created tasks from the second subagent's report. If any task's `for-agent` tag does not match what the breakdown marked, or a task failed to be created, say so plainly.

The skill is finished once the report is shown; stop there.
