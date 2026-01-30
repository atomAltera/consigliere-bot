we have to build a telegram bot which helps to collect participants to the event. 
the logic is relatively simple:

  - we have two event in a week: monday and saturday
  - the day before each event we post a poll in our telegram group
  - poll has title and several options: 
    1. will come at 19:00
    2. will come at 20:00
    3. will come at 21:00 or later
    4. will decide later
    5. will not come
  - participants are voting in the poll
  - poll is not anonymous
  - each member can vote only once and choose only one option
  - members can retract their vote
  - members can retract their vote and revote
  - at noon of event day we pin the poll with flag "notify all members" and post additional updatable message which contains information about the event and the poll results (results message)
  - if members changes votes after results message is posted, we update the results message
  - if we can't collect enough participants (minimum 11) before 5pm, we post message that event is canceled
  - if we see that we need 1-2 members before 5pm, we post and pin message that we are lookling 1 or 2 herous to save the day

for now let's focus only at poll and results message. bot has to post poll and read it results. for not let's focus on manual triggers and forget about automation. so, we have to have the following actions:
  - post poll for specific day (date)
  - have a list of all active polls (mostly only one)
  - pint the poll and notify all members
  - poit results message
  - we have somehow track voters.
  - if results message is posted we have to keep it up to date. so we have to have an action to update the results message. this action may be autotriggered if there comes an update of the poll from telegram.
  - i want to control all this staff with bot commands. for example:
    - `/poll <day>` to post poll for a specific day. 
    - `/results` to post results message
    - `/cancel` to delete results message and post cancalation message
    - `/pin` to pin results message
important part here: only admins of chat can use this command. also, after command is fullfiled, message has to be deleted by bot. so, i do not want that commnds stay in chat history.
we have to store somewhere all ongoing polls (/result, /cancel and /pin affects the last one created poll). and also we have to store all voting history in database for each poll. so we have to be able to build results list for each poll.
