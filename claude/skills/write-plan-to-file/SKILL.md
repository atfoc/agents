---
name: write-plan-to-file 
description: Writes plan to file
disable-model-invocation: true
---

# Write plan to file
Write the whole plan with all updates and resolutions of open questions to file. The file is ./.tasks/{taskName}/plan.md.
Infer task name from the plan itself.
As output tell user plan is writen to  {path}. If you want to implement it run in new session /implement-plan {path} 
