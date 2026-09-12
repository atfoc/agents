# Spec to tasks with two agents

Give the implementation spec to one subagent, which splits it into vertical slices by
`split-work-into-vertical-slices.md` and hands back every slice with its title, body and blockers.
Then give those slices to a second subagent, which creates one task per slice, verbatim, wherever
the user asked the tasks to go, every one of them tagged `for-agent` — the tag
`implementing-tasks-workflow.md` pulls by, so an untagged task is never picked up. A store with no
way to tag a task is a stop for the second agent, never a reason to create the tasks untagged. If
the user did not say where tasks go, ask before spawning anything; if the split reports a stop or a
conflict, report it and never spawn the second agent.
