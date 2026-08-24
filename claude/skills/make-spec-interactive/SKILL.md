---
name: make-spec-interactive
description: Used when user requests interactive session for creating implementation spec
disable-model-invocation: true
---

# Make implementation spec in iteractive session
The goal is to provide user with interactive session. The output of this session is implementation spec.

Implementation spec is fully detailed set of instructions what needs to be changed and where. Change is adding removing or updating code or resources.
Minimum level of details required here is for example signature of functions, classes definition (fields or methods), interfaces and who is implementing them.
Desired level of details is also at least pseudo code of implementation of new functinos or additions to existing functions. If function includes updating as combination
of deleting and adding then opt for full rewrite in pseduo code then explaing how to modify existing. If some method, class, interface or similar is mentioned 
then it also needs to be define in details explained above. There is no leaving room for interpertatino of implementation based on name from pseudo code for example.
This detailed spec for each item needs to be followed with location. For new methods classes and similar there needs to be information where they are placed.
For changing methods, classes, etc location about where orginal method is. 

# How to arrie to implementation spec
Process to follow in order to get implementation spec is the following. First feature is defined, then that is converted to detailed implementation spec. User stays 
in the loop providing feedback and instructions. More info on that.

## Feature definition 
This is not on implementation level but in a high level description what needs to be built. Not from technical perspective but from product / user perspective. 
Sometimes that will be from technical perspective as well. Usually for refactors, reusable components etc. This do not have feature definition maybe going back to buisnis 
or users but they will have a clear definition what needs to be built without going to implementation details explained above. An example could look like this

The social element of liking and disliking user post is beeing added to our platform. Like and dislike buttons with a progress
bar representing ratio of likes and dislikes is added bellow user post title. Clicking on like button will flash a animation ...

At this point there is no talk about implementation detials like signatures pseudo code etc.



## Going from feature definition to implementation

# Other important details
- Do not output anything to a file until user request that
