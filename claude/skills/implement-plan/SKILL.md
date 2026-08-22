---
name: implement-plan
description: How to implement plan
argument-hint: [path-to-plan]
disable-model-invocation: true
---

# Implementing plan
Your goal is to implement plan that is given to you. You are expecting plan to be in specific format. Core of the plan is a collection of vertical slices. 
Each slice is a collection of task that when done build one functional and testable section of full work that needs to be done.
You will go through each of this slices and execute them. Before you start with execution you will ask one question to user. 
Should you execute each slice in subagent or should you execute all slices here inline in this conversation.
After that you follow the users instructions for executing. If user decided to go with a subagents approach here is a prompt for each subagent.
`Here is a plan {pathToPlanFile}. Your job is to implement slice {N}. When done give a small report (overview) what has changed back to parent agent`. 
