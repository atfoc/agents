---
name: make-feature-definition
description: Used when user requests interactive session for creating feature definition
argument-hint: [what is being built]
disable-model-invocation: true
---

# Make feature definition in interactive session
Your goal is to provide user with interactive session. The output of this session is feature definition.
Every question you ask here is about what is being built, never about how it will be implemented.

## Feature definition 
This is not on implementation level but in a high level description what needs to be built. Your goal is to come to shared understanding what is beeing built not
how. So you should not focus on implementation level questions like what to do with this class etc, but on questinos that help you understand scope, reason, direction and goal.
Even when feature definition is more implementaiton in nature like refactoring a class.
When dealing with technical feature definitinos like refactors you should still not ask me implementation questions.
An example could look like this

The social element of liking and disliking user post is beeing added to our platform. Like and dislike buttons with a progress
bar representing ratio of likes and dislikes is added bellow user post title. Clicking on like button will flash a animation ...

At this point there is no talk about implementation detials like signatures pseudo code etc.

## How to build feature definition
If I does not provide at least some description of what should be built, then start with a question what is beeing built.
When there is at least some information about what we are building we start the loop of building feature desing. This loop is designed to keep me informed about descisions beeing made.
Your job here is to ask me questions in order for us to build shared understanding and align on what feature is. In doing so we have made feature design. Important thing is that you are in 
charge to cover all cases. If you imagine feature design as graph and we are at the root trying to discover and build the whole graph your job is to make sure we cover all potential paths.
When asking a questions always provide one recommendation for anwser. Besides design questions about what we are building it is important for us to cover how does this feature impact existing project.
We need to discuss about how it changes, adds or removes things from project. Now we are talking about functionality not yet going into implementatino details. Make sure you raise any conflicts 
or problems that this new feature could impact our current project. You can recommend when to stop but it is my decsision do we keep going or we are moving to next step.
When we are finished first write out the whole feature design so that we can one more time check are we in sync.

## More on asking questions
When asking questions follow some natural order do not jump from topic to topic. We start with one topic and we drill down before we change the topic.
That way we build always on what was previously decided until we need to change topic because we covered everything.

# After we are done
When feature definition is done we write it to ./.tasks/{taskName}/feature-design.md .
You will infer task name based on conversation. Do not write this file until we finish the feature definition completly, and I approve the write. The reason for this it is harder to continuesly edit a file
instead of our inline conversation.
After the file is written report the path and stop. This skill does not produce implementation spec.

# Other important details
- Do not output anything to a file until user request that
- Do not use ask question tool
- When repeating questions, when in my anwser I did not cover questions do not reword them repeat them verbatium. Only reword it if I ask for clarification or reword
