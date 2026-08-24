---
name: write-plan-to-file
description: Write the plan produced in this conversation, with all updates and resolved open questions, to ./.tasks/{taskName}/plan.md. Use when the user asks to save, write, or persist the plan to a file — "write the plan to file", "save this plan".
disable-model-invocation: true
---

# Write plan to file
Write the whole plan with all updates and resolutions of open questions to file. The file is ./.tasks/{taskName}/plan.md.
Infer task name from the plan itself.
The plan is whatever the user pointed at: the plan made in this conversation, or a plan they pasted or named. If there is no plan, ask which plan to write and stop.
As output tell user plan is writen to  {path}. If you want to implement it run in new session /implement-plan {path}
