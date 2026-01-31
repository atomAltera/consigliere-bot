package bot

// User error messages (user mistakes, shown directly)
const (
	MsgInvalidDateFormat  = "Invalid date format. Use day name (e.g., monday, sat) or YYYY-MM-DD"
	MsgPollAlreadyExists  = "There is already an active poll in this chat. Cancel it first with /cancel before creating a new one."
	MsgNoActivePoll       = "No active poll found"
	MsgPollMessageMissing = "Poll message not found"
	MsgInvalidUsername    = "Invalid username"
	MsgVoteUsage          = "Usage: /vote @username <option 1-5>\nOptions: 1=19:00, 2=20:00, 3=21:00+, 4=decide later, 5=not coming"
	MsgInvalidVoteOption  = "Invalid option. Use 1-5:\n1=19:00, 2=20:00, 3=21:00+, 4=decide later, 5=not coming"
)

// System error messages (internal errors, hide details from user)
const (
	MsgInternalError            = "An internal error occurred. Please try again later."
	MsgFailedCreatePoll         = "Failed to create poll. Please try again."
	MsgFailedRenderPollTitle    = "Failed to render poll title. Please try again."
	MsgFailedSendPoll           = "Failed to send poll. Please try again."
	MsgFailedSavePoll           = "Failed to save poll. Please try again."
	MsgFailedGetResults         = "Failed to get results. Please try again."
	MsgFailedRenderTitle        = "Failed to render title. Please try again."
	MsgFailedRenderResults      = "Failed to render results. Please try again."
	MsgFailedSendResults        = "Failed to send results. Please try again."
	MsgFailedSaveResults        = "Failed to save results. Please try again."
	MsgFailedPinPoll            = "Failed to pin poll. Please try again."
	MsgFailedSavePollStatus     = "Failed to save poll status. Please try again."
	MsgFailedSendCancellation   = "Failed to send cancellation message. Please try again."
	MsgFailedRecordVote         = "Failed to record vote. Please try again."
)

// Format strings for dynamic messages
const (
	MsgFmtEventCancelled = "⚠️ Event on %s has been cancelled"
	MsgFmtVoteRecorded   = "Recorded vote for %s: %s"
)
