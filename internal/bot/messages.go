package bot

// User error messages (user mistakes, shown directly)
const (
	MsgInvalidDateFormat  = "Неверный формат даты. Используйте название дня (например, понедельник, сб) или ГГГГ-ММ-ДД"
	MsgPollAlreadyExists  = "В этом чате уже есть активный опрос. Сначала отмените его командой /cancel"
	MsgNoActivePoll       = "Активный опрос не найден"
	MsgNoCancelledPoll    = "Нет отменённых опросов"
	MsgPollDatePassed     = "Нельзя восстановить опрос для прошедшей даты"
	MsgPollMessageMissing = "Сообщение с опросом не найдено"
	MsgInvalidUsername    = "Неверное имя пользователя"
	MsgVoteUsage          = "Использование: /vote @имя <опция 1-5>\nОпции: 1=19:00, 2=20:00, 3=21:00+, 4=решу позже, 5=не приду"
	MsgInvalidVoteOption  = "Неверная опция. Используйте 1-5:\n1=19:00, 2=20:00, 3=21:00+, 4=решу позже, 5=не приду"
)

// System error messages (internal errors, hide details from user)
const (
	MsgInternalError          = "Произошла внутренняя ошибка. Попробуйте позже."
	MsgFailedCreatePoll       = "Не удалось создать опрос. Попробуйте ещё раз."
	MsgFailedGetPoll          = "Не удалось получить опрос. Попробуйте ещё раз."
	MsgFailedRenderPollTitle  = "Не удалось сформировать заголовок опроса. Попробуйте ещё раз."
	MsgFailedSendPoll         = "Не удалось отправить опрос. Попробуйте ещё раз."
	MsgFailedSavePoll         = "Не удалось сохранить опрос. Попробуйте ещё раз."
	MsgFailedCancelPoll       = "Не удалось отменить опрос. Попробуйте ещё раз."
	MsgFailedRestorePoll      = "Не удалось восстановить опрос. Попробуйте ещё раз."
	MsgFailedGetResults       = "Не удалось получить результаты. Попробуйте ещё раз."
	MsgFailedRenderTitle      = "Не удалось сформировать заголовок. Попробуйте ещё раз."
	MsgFailedRenderResults    = "Не удалось сформировать результаты. Попробуйте ещё раз."
	MsgFailedSendResults      = "Не удалось отправить результаты. Попробуйте ещё раз."
	MsgFailedSaveResults      = "Не удалось сохранить результаты. Попробуйте ещё раз."
	MsgFailedPinPoll          = "Не удалось закрепить опрос. Попробуйте ещё раз."
	MsgFailedSavePollStatus   = "Не удалось сохранить статус опроса. Попробуйте ещё раз."
	MsgFailedRenderCancellation = "Не удалось сформировать сообщение об отмене. Попробуйте ещё раз."
	MsgFailedSendCancellation   = "Не удалось отправить сообщение об отмене. Попробуйте ещё раз."
	MsgFailedRecordVote       = "Не удалось записать голос. Попробуйте ещё раз."
)

// Format strings for dynamic messages
const (
	MsgFmtEventCancelled = "⚠️ Игра %s отменена"
	MsgFmtVoteRecorded   = "Записан голос за %s: %s"
)
