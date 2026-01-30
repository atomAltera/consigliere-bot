package poll

import "time"

type Vote struct {
	ID            int64
	PollID        int64
	TgUserID      int64
	TgUsername    string
	TgFirstName   string
	TgOptionIndex int
	VotedAt       time.Time
}

func (v *Vote) OptionKind() OptionKind {
	return OptionKind(v.TgOptionIndex)
}

func (v *Vote) DisplayName() string {
	if v.TgUsername != "" {
		return "@" + v.TgUsername
	}
	return v.TgFirstName
}
