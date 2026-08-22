---
name: make-plan
description: Make a plan for implementing given spec
disable-model-invocation: true
---

# Make a plan
This skill is used to conver spec to plan. Spec is a document that is describing desired state in current project. Plan is set of instructions how to get to there from current state.

# What plan contains
Plan should have the following sections:
- Highlevel overview of current state
- Highlevel overview of steps to take us to desired step
- Collection of vertical slices
- Open questions and blockers

## Highlevel overview of current state
The goal of this step is to make it easier to read the rest of the plan. We should analyze current project and present relevant things for the rest of the plan.
We should highlight what components exist, what is their job, how do they interact with each other. We should only highlight components that are later going to be mentioned
in plan for some reason. It can be because we will change them or replace them, or build a new one because old components do not satisfy some requirements.

## Highlevel overview of steps to take 
Here we want to connect to what was previously said. We want to define and explain what will we change why and how. What will we add, what will we remove, what will we chagne.
At this as on previous level we are talking on level of modules and public apis, with maybe side of persistence or similar chagnes.
We are not talking about implementatioo details. Goal is to build a map of all components their interaciton and how those changes will be changed, so that when we get in more low level details we
know how they reflect to higher level picture.

## Vertical slices
This section is core of a plan. Whole work needed to be done is split into what we call vertical slices. 
Vertical slice is set of tasks that when completed build one functional and testable section of whole work that needs to be completed. 
We define vertical slice by starting at the top level. This is a place usually some interaction happens or information in flow of information starts.
For example if we are talking about backend this could be a http handler, that qualifies as interaction or source of information. 
We could also talk about some public api method on some module. From there we start going deeper layer per layer, defing work that needs to be done at each level.
Important thing is for one slice is to flow as linear as possible and make as few branchings as possible. Techniques we use for cutting of branching is mostly stubbing. For example
we define method and call it but we do not implement it, that is left to the other slice.
Slices should highlith what will be added, removed or changed.
Final part of the slice will be verification. Set of facts that need to be verified in order to consider slice complete. This will usualy translate to unit tests.

### Example of vertical slice
We have a add to cart handler. This handler recieves reqeust from client. Here we would show example of request and/or schema. Then handler verifies request and build domain object.
UserShopigCartService service method addToCart is invoked given domain request. This handler then fetches users cart from database with lock using repository method getUserCartLock.
Item id is verified is it already in cart. If it is in cart request is failed. If it is not in cart it is added to cart. After adding to the cart user cart is persisted to database.
All of this is wraped in transaction, and on commit hook we are going to be tracking following analytics events. We are not going to be implementing events here that will be done in seprate slice, we are
just going to leave empty methods here for tracking. 

Note that in this example we did not talk about changes to databse schema or tracking events. Both of this could belong to this slice, but we are trying to keep slices focues. If we introduced both we 
are making branching of slice and in that increasing scope. So that is why descision here was to leave implementatio of tracking in seperate slice possibliy grouping bunch of or all events thogether.
And for database there could be much more work like mappers from json columns or adding more then one column, that is all left as seprate slice. If we here were just adding one or two methods in repository in order
to server this handler then we would add those task to this slice. And still if we had that we would add it to this slice, just changes to schema or migration effort would be left as second slice.

## Open questions and blockers
This is section where we write all things we could not define in the plan. It can be contradicting facts in the spec itself. It could be contradicting facts in spec and real code.
For questions we do not have information on and we can not decide we leave to user to make a choice. 
For blockers anythign that is not creating plan but could help enrich plan with details and clear uncertanties we wirte here. Example is if spec require us to invoke kubctl binary. We might not know exact 
arguments and what they return. So open question / blocker could be exploration of that so that we can better define those details in plan.

# What after we output plan
When we output a plan we let user read it and understand it. Users then ask us follow ups or to explain plan in more details. Maybe we correct some parts of the plan, or anwser and close open questions.
We wait until user says to do something different, until then we are in loop of ansering questions about the plan and updating the plan.
When updating the plan our anwser is not whole plan updated, but we anwser with diffs to the plan. What has the required update changed in the plan.

# Output format
For the output format we focus on taking user from high level of abstraction to lower level of abstraction, we do this for each section. We try to keep lenght of our anwser short and direct.
We alwasy provide users with overview of topics before we dive into details of each section of the topic. And we always use appropreat output for the job. For example we use ascii diagrams for ui
or flows or architectural discussions. But never really only on diragrams they are always followed with text explanations.

# Other key details
- We do not write or plan integration tests, or end to end tests.
- We always write unit tests for changes.
- We never leave in plan `We will do x or y`. We always choose x or y and define it in plan.
- We do not run commands in root `/`. They are slow and almost always provide little to no value. Always foucs commands on specific directories. Mostly you will find everythign you need in cwd.
- We are not making any code changes, this whole process is readonly.


