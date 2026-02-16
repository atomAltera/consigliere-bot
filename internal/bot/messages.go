package bot

// Permission messages
const (
	MsgChatNotPermitted = "Этот чат не зарегистрирован для использования бота"
)

// User error messages (user mistakes, shown directly)
const (
	MsgInvalidDateFormat  = "Неверный формат даты. Используйте название дня (например, понедельник, сб) или ГГГГ-ММ-ДД"
	MsgPollAlreadyExists  = "В этом чате уже есть активный опрос. Сначала отмените его командой /cancel"
	MsgNoActivePoll       = "Активный опрос не найден"
	MsgNoPoll             = "Опрос не найден"
	MsgNoCancelledPoll    = "Нет отменённых опросов"
	MsgPollDatePassed     = "Нельзя восстановить опрос для прошедшей даты"
	MsgPollMessageMissing = "Сообщение с опросом не найдено"
	MsgInvalidUsername    = "Неверное имя пользователя"
	MsgVoteUsage          = "Использование: /vote @имя <опция 1-5>\nОпции: 1=19:00, 2=20:00, 3=21:00+, 4=решу позже, 5=не приду"
	MsgInvalidVoteOption  = "Неверная опция. Используйте 1-5:\n1=19:00, 2=20:00, 3=21:00+, 4=решу позже, 5=не приду"
	MsgNoUndecidedVoters  = "Нет участников, которые ещё не определились"
	MsgNotEnoughPlayers   = "Недостаточно игроков. Нужно минимум 11 человек на 19:00 и 20:00"
	MsgInvalidStartTime   = "Неверный формат времени. Используйте: /done 19, /done 20:00, /done 21:30"
	MsgNickUsage     = "Использование: /nick @username игровой_ник [пол]\nНик в кавычках если с пробелами: /nick @user \"Мадам Жу\"\nПол (опционально): м/ж/m/f/д"
	MsgNickDuplicate = "Такая связка уже существует"
	MsgInvalidGender = "Неверный пол. Используйте: м/ж/m/f/д"
)

// System error messages (internal errors, hide details from user)
const (
	MsgInternalError            = "Произошла внутренняя ошибка"
	MsgFailedCreatePoll         = "Не удалось создать опрос"
	MsgFailedGetPoll            = "Не удалось получить опрос"
	MsgFailedRenderPollTitle    = "Не удалось сформировать заголовок опроса"
	MsgFailedSendPoll           = "Не удалось отправить опрос"
	MsgFailedSavePoll           = "Не удалось сохранить опрос"
	MsgFailedCancelPoll         = "Не удалось отменить опрос"
	MsgFailedRestorePoll        = "Не удалось восстановить опрос"
	MsgFailedGetResults         = "Не удалось получить результаты"
	MsgFailedRenderTitle        = "Не удалось сформировать заголовок"
	MsgFailedRenderResults      = "Не удалось сформировать результаты"
	MsgFailedSendResults        = "Не удалось отправить результаты"
	MsgFailedSaveResults        = "Не удалось сохранить результаты"
	MsgFailedPinPoll            = "Не удалось закрепить опрос"
	MsgFailedSavePollStatus     = "Не удалось сохранить статус опроса"
	MsgFailedRenderCancellation = "Не удалось сформировать сообщение об отмене"
	MsgFailedSendCancellation   = "Не удалось отправить сообщение об отмене"
	MsgFailedRenderRestore      = "Не удалось сформировать сообщение о восстановлении"
	MsgFailedSendRestore        = "Не удалось отправить сообщение о восстановлении"
	MsgFailedRecordVote         = "Не удалось записать голос"
	MsgFailedGetUndecided       = "Не удалось получить список неопределившихся"
	MsgFailedRenderCall         = "Не удалось сформировать сообщение"
	MsgFailedSendCall           = "Не удалось отправить сообщение"
	MsgFailedRenderCollected    = "Не удалось сформировать сообщение о наборе"
	MsgFailedSendCollected      = "Не удалось отправить сообщение о наборе"
	MsgFailedSaveNick           = "Не удалось сохранить ник"
	MsgFailedRefresh            = "Не удалось обновить сообщения"
)

// Format strings for dynamic messages
const (
	MsgFmtEventCancelled   = "⚠️ Игра %s отменена"
	MsgFmtVoteRecorded     = "Записан голос за %s: %s"
	MsgFmtNickCreated      = "Ник сохранён: %s → %s"
	MsgFmtNickCreatedByID  = "Ник сохранён: ID %d → %s"
)
