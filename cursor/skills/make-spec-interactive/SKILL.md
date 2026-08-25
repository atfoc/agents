---
name: make-spec-interactive
description: Used when user requests interactive session for creating implementation spec
disable-model-invocation: true
---

# Make implementation spec in iteractive session
Your goal is to provide user with interactive session. The output of this session is implementation spec.

Implementation spec is fully detailed set of instructions what needs to be changed and where. Change is adding removing or updating code or resources.
Minimum level of details required here is for example signature of functions, classes definition (fields or methods), interfaces and who is implementing them.
Desired level of details is also at least pseudo code of implementation of new functinos or additions to existing functions. If function includes updating as combination
of deleting and adding then opt for full rewrite in pseduo code then explaing how to modify existing. If some method, class, interface or similar is mentioned 
then it also needs to be define in details explained above. There is no leaving room for interpertatino of implementation based on name from pseudo code for example.
This detailed spec for each item needs to be followed with location. For new methods classes and similar there needs to be information where they are placed.
For changing methods, classes, etc location about where orginal method is. Another part of implementatino spec is how do we test and verify things work.
We need to define test cases for each unit it makes sense.

# How to create implementation spec
In order to create implementation spec you will follow this process. First we need to define feature. Then we need to  convert it to detailed implementation spec. I stay 
in the loop providing feedback and instructions. More info on that.

## Feature definition 
This is not on implementation level but in a high level description what needs to be built. Not from technical perspective but from product / user perspective. 
Sometimes that will be from technical perspective as well. Usually for refactors, reusable components etc. This do not have feature definition maybe going back to buisnis 
or users but they will have a clear definition what needs to be built without going to implementation details explained above. An example could look like this

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

## Going from feature definition to implementation
Here we will repeat similar approach as for creating feature design. Just instead of having loop of questions and anwsers about functionality that we are building, we are discussing how to implement that 
in our current project. We start at going through whole feature design and talking about how each part will be implemented. Your job again is to make sure we cover everytihng. Do not assume anything 
we need to build shared understanding of the whole implementation. Here besids approaches code split in modules we are discussing stuff like migraitons, signature of methods etc. We need to disucss
what and how is beeing added, removed or changed in code base. You will same as in feature design ask me a question and provide one recommendation. Same as in feature design you can recommend when to end
but it is my call we do not continue until I say so. 

When we are done before we continue you will write the full implementation spec back to me so we can one more time check are we in sync.

## More on asking questions
When asking questions follow some natural order do not jump from topic to topic. We start with one topic and we drill down before we change the topic.
That way we build always on what was previously decided until we need to change topic because we covered everything.

# After we are done with designing
When both feature design and implementation spec is done we write both to ./.tasks/{taskName}/feature-design.md and ./.tasks/{taskName}/implementation-spec.md .
You will infer task name based on conversation. Do not write this files until we finish designing both of them completly, and I approve the write. The reason for this it is harder to continuesly edit files
instead of our inline conversation.

# Other important details
- Do not output anything to a file until user request that
- Do not use ask question tool
- When repeating questions, when in my anwser I did not cover questions do not reword them repeat them verbatium. Only reword it if I ask for clarification or reword

Stop once both files are written to ./.tasks/{taskName}/, or earlier if I say we are done.
