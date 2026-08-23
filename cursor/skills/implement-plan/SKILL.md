---
name: implement-plan
description: Implement a plan file written as a collection of vertical slices — read the slices, ask whether to run them inline or in subagents, then execute them in order. Use whenever the user points at a plan file and asks to implement it, execute it, or build it.
disable-model-invocation: true
---

# Implementing plan
Your goal is to implement plan that is given to you. The plan is whatever the user pointed at: a path in this conversation, a file they opened, or the skill argument. If there is no plan path, ask which plan file to implement and stop.

You are expecting plan to be in specific format. Core of the plan is a collection of vertical slices. 
Each slice is a collection of task that when done build one functional and testable section of full work that needs to be done.
You will go through each of this slices and execute them. Before you start with execution you will ask one question to user. 
Should you execute each slice in subagent or should you execute all slices here inline in this conversation.
After that you follow the users instructions for executing. If user decided to go with a subagents approach here is a prompt for each subagent.
`Here is a plan {pathToPlanFile}. Your job is to implement slice {N}. When done give a small report (overview) what has changed back to parent agent`. 

If you are already running as a subagent you cannot spawn further subagents — nesting stops after one level. Say so, execute all slices inline, and continue.
