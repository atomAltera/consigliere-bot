# TODO

here a are some todos has to be done for the current project. 

general project info is placed in ./task.md

!!! HOW TO WORK !!!
- you pick only one, the most relevant and not completed task from the todo list below.
- IF ALL TASKS ARE DONE OUTPUT "ALL TODO ITEMS ARE DONE" AND DO NOTHING MORE
- Focus on chosen task and implement it.
- mark task as completed.
- if you failed to finish the task, print to output message "FAILED TO DO THE TASK" and put details
- run review before commit, commit you run only when all other tasks are done.
- if review found some issues, you do not mark this item as done. you run review only when other tasks except commit are done.
- to review made changes run `codex --full-auto review` and add flag which makes codex print review and exit.
- commit after task is done

## TODO LIST

- [x] each command (except /poll) works on last active poll. if there is not last active poll in this exact chat, print error message (original message from user with command has to be removed normally)
- [x] when placing new poll (/poll command) make sure there is no other active poll at this exact chat, write error message if any other active poll exists
- [x] /result command now sends results message after each call, but only last result message get's update by changes in poll. make /results command delete all previos results messages before creating a new one.
- [x] make /poll command be more versatile, if no argumants passed, make poll to nearest monday or saturday. also it should accept day of week as argument.
- [x] i do not like that while handling errors commands.go (and may be in other places too, i did not dig deeper) we swallow original error. actually we send it to user to chat, but it is wrong behaviour, we have to develop a common wrapper for that cases, in case of error we notify user about error, if it is some system error we notify it that internal error happened, but we must write original error to log. descide yourdelf how to do it in most convinient and go idiomatic way.
- [x] poll title has to be in templates too (same as results), it would be massive in future.
- [x] we should have commands to manualy add/remove users from poll results event if they are not voted yet or voted with different option. for example, user @bob may vote for not to come, but later in direct message he told me that he will come at 20:00 and i have to have facility to add him to results. i want to have command like `/vote @bob <poll option number (starting from 1)>. this votes have to have special flag but logicaly it should be treated as user changed his mind and actually voted/revote with different option.
- [x] /cancel has to unpin current pinned poll if any. 
- [x] /pin has to unpin all previously pinned polls if any before pinning current one
- [x] i do not like Status fields of Poll entity. let's get rid of it and instead let's add two bool fields: IsActive and IsPinned. write a simple migration sql script too which adds two columns, that updates rows, then drops status column. make sure it would be idenotent
- [x] we do not need to list all users which skips event in result, instead we can show only count of them
- [x] /pin command must not write report about "Poll pinned successfully"
- [x] /pin command must notify all users in chat that message was pinned (if available for bots, if not complete this task as add comment this is it impossible)
