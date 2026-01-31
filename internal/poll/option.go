package poll

type OptionKind int

const (
	OptionComeAt19 OptionKind = iota
	OptionComeAt20
	OptionComeAt21OrLater
	OptionDecideLater
	OptionNotComing
	OptionRetracted OptionKind = -1
)

var optionLabels = []string{
	"Приду к 19:00",
	"Приду к 20:00",
	"Приду к 21:00 или позже",
	"Решу позже",
	"Не приду",
}

func (o OptionKind) Label() string {
	if o < 0 || int(o) >= len(optionLabels) {
		return "Неизвестно"
	}
	return optionLabels[o]
}

func (o OptionKind) IsAttending() bool {
	return o >= 0 && o <= OptionComeAt21OrLater
}

func AllOptions() []string {
	return optionLabels
}
