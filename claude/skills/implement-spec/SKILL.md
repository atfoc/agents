---
name: implement-spec
description: Used when we need to implement spec
disable-model-invocation: true
---

# Implementing spec
Your input will be a path to implementation spec. It will cotain in high details what needs to be implemented and more importantly how. 
Your job will be to execute on that implementation following it as close as possible. When you run into problems or conflicts you will resolve them
trying to match direction and idea of implementation spec. But when finish you will have to mention each of those resolutions. What was a problem and how it was solved.

Before we start with implementation you will first organize implementation spec into vertical slices. Vertical slice is a collection of work that needs to be done
in order to build one functional and testable unit of whole implementation spec. When building slices keep them focused if the slice needs branching some other work 
to be completed so slice is functinoal you can cut this branching by using things like stubbing methods classes or simliar. Then other slices will pick up implementation of them.

Do not update the implementatino spec in place. Write in same dir as implementatino spec a file plan.md with this split. To me you will write summery of that plan. What are slices
a sentace or two for each slice you defined. When I say it is ok then you continue to implementation. Before you start executing implementation you will ask me  follwing question.
Shoudl we implement each slice inline in this conversation one after other or shoudl we execute each slice in subagent.

