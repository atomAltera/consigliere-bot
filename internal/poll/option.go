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
	"Will come at 19:00",
	"Will come at 20:00",
	"Will come at 21:00 or later",
	"Will decide later",
	"Will not come",
}

func (o OptionKind) Label() string {
	if o < 0 || int(o) >= len(optionLabels) {
		return "Unknown"
	}
	return optionLabels[o]
}

func (o OptionKind) IsAttending() bool {
	return o >= 0 && o <= OptionComeAt21OrLater
}

func AllOptions() []string {
	return optionLabels
}
