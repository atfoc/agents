---
name: make-implementation-spec
description: Used when user requests interactive session for creating implementation spec from a feature definition
disable-model-invocation: true
---

# Make implementation spec in interactive session
Your goal is to provide user with interactive session. The output of this session is implementation spec.

The input is a path to a feature definition, or a feature definition produced earlier in this conversation. If there is neither, stop and ask for one.
Never invent a feature definition and never start from a bare description.

Implementation spec is fully detailed set of instructions what needs to be changed and where. Change is adding removing or updating code or resources.
Minimum level of details required here is for example signature of functions, classes definition (fields or methods), interfaces and who is implementing them.
Desired level of details is also at least pseudo code of implementation of new functinos or additions to existing functions. If function includes updating as combination
of deleting and adding then opt for full rewrite in pseduo code then explaing how to modify existing. If some method, class, interface or similar is mentioned 
then it also needs to be define in details explained above. There is no leaving room for interpertatino of implementation based on name from pseudo code for example.
This detailed spec for each item needs to be followed with location. For new methods classes and similar there needs to be information where they are placed.
For changing methods, classes, etc location about where orginal method is. Another part of implementatino spec is how do we test and verify things work.
We need to define test cases for each unit it makes sense.

## Going from feature definition to implementation
Here we will repeat similar approach as when the feature definition was created. Just instead of having loop of questions and anwsers about functionality that we are building, we are discussing how to implement that 
in our current project. We start at going through whole feature design and talking about how each part will be implemented. Your job again is to make sure we cover everytihng. Do not assume anything 
we need to build shared understanding of the whole implementation. Here besids approaches code split in modules we are discussing stuff like migraitons, signature of methods etc. We need to disucss
what and how is beeing added, removed or changed in code base. You will same as in feature design ask me a question and provide one recommendation. Same as in feature design you can recommend when to end
but it is my call we do not continue until I say so. 

When we are done before we continue you will write the full implementation spec back to me so we can one more time check are we in sync.

## More on asking questions
When asking questions follow some natural order do not jump from topic to topic. We start with one topic and we drill down before we change the topic.
That way we build always on what was previously decided until we need to change topic because we covered everything.

# After we are done
When implementation spec is done we write it to implementation-spec.md in the same directory as the feature definition it was given.
Do not write this file until we finish the implementation spec completly, and I approve the write. The reason for this it is harder to continuesly edit a file
instead of our inline conversation.
After the file is written report the path and stop. This skill does not split the spec into tasks and does not implement it.

# Other important details
- Do not output anything to a file until user request that
- Do not use ask question tool
- When repeating questions, when in my anwser I did not cover questions do not reword them repeat them verbatium. Only reword it if I ask for clarification or reword
