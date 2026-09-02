# Spec to tasks with two agents

Give the implementation spec to one subagent, which splits it into vertical slices by
`split-work-into-vertical-slices.md` and hands back every slice with its title, body and blockers.
Then give those slices to a second subagent, which creates one task per slice, verbatim, wherever
the user asked the tasks to go. If the user did not say where tasks go, ask before spawning
anything; if the split reports a stop or a conflict, report it and never spawn the second agent.
