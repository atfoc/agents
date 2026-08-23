---
name: explain-code
description: Explain code at the abstraction level the user asked for — a function, a class, a module, a whole codebase, or how something is used — and keep leading them through the codebase as they zoom in and out. Use when the user asks what some code does, how it works, or where and how it is used.
disable-model-invocation: true
---

# Explain code
User will provide you with input what code they need to be explained. Your job is to build understanding of the code and respond with explanation.

# How to responde 
Your response need to be different dependent on what user asks you. Main difference comes down from abstraction level user ask for. General idea is if user asks how method works anwser needs to be different from when they ask how does this module work. Here are some guidelines

## For functions
This is a low level explanation. You need to dig into implementation details. First start with explaining signature, what it takes as inputs, what it returns.
For each input explain what are the valid inputs and what inputs lead to failure. Then explain response. If response is straith forward one sentece is enough.
If response has different states depending on something then they all should be covered in explanation. For example for return type could be, function returns a user object fetched from database by filters provided by arguments. For different states of response it could be. Function returns a object that can be in the following states, user not found, user banned, user active, then explaining for each what data it carries with them selfs.

Special case is when you are explaining algorithms. There besides what is explained in previous paragraph you also add examples of algorithm input, output, and maybe break down step by step how it got there (this could be depending on algorithm compexity). You should cover at least one example from each case, and one case from each edge case.
## For classes
Here you should focus first on public api of the class. You should explain each public function in class, what it does and any special rules if it has how it should be used. 
You are not going to be focusing on inputs and outputs, but more on what does that collection of functions achive. Is there any specific state that this classe explains or encapsulates. For example if class is modeled in such way to represent some state machine it should explain what that state machine is.
## For modules (collection of classes)
Here you are focusing more on what collection of classes and what they are supposed to do. Each class should have a sentance or two what is its job and what it encapsulates.
You are not going in public api for each class.
## Whole codebase
This is a rear case, only if codebase is quite small, then it is same as explaining for modules. If codebase is small contains 2 to 3 modules then you can repeat the for moudles explanation. If it has more then that, then you should just show what are those modules one sentance about each, and ask user in which they want to dive in.
## For (function) usage 
Sometimes user will not ask you to explain function or module but how it is used. Then your job is to combine knowladge about how function works with how it is used, in order
to show whole picture to user. For how function is used you should travers its call stack all the way until you reach top levels. Each new level will unlock more high level information about each usecase. The goal is to define each usecase as clear and as close to product as possible. So explanation like this function fetches user from database in locked state and it is used when ever user is mutaated. It should output more something like. This function returns locked user from database and it is used in following usecases. Then sentance or two about each usecase, when user tries to buy gem stones then they object for state is fetched from db locked and stash is updated for the right amount.

# What next
Depending on what level user started, we are going to be in loop and repeating this explanations for what user chooses to zoom in or out. If User asked about function then they
might ask to look where it used and how, that might lead them to other module and they might ask to explain that module. Then they could drill down some class and then again to some method. Your job is to using this rules progresivly disclose information and complexity and in interactive way lead user through codebase.
