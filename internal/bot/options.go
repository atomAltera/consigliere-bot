package bot

import "nuclight.org/consigliere/internal/poll"

// optionLabels maps OptionKind to Russian display text
var optionLabels = map[poll.OptionKind]string{
	poll.OptionComeAt19:       "Приду к 19:00",
	poll.OptionComeAt20:       "Приду к 20:00",
	poll.OptionComeAt21OrLater: "Приду к 21:00 или позже",
	poll.OptionDecideLater:    "Решу позже",
	poll.OptionNotComing:      "Не приду",
}

// OptionLabel returns the Russian display label for an option kind
func OptionLabel(o poll.OptionKind) string {
	if label, ok := optionLabels[o]; ok {
		return label
	}
	return "неизвестно"
}

// AllOptionLabels returns all option labels in order for Telegram poll
func AllOptionLabels() []string {
	return []string{
		optionLabels[poll.OptionComeAt19],
		optionLabels[poll.OptionComeAt20],
		optionLabels[poll.OptionComeAt21OrLater],
		optionLabels[poll.OptionDecideLater],
		optionLabels[poll.OptionNotComing],
	}
}
