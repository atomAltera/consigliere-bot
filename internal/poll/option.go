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

func (o OptionKind) IsAttending() bool {
	return o >= 0 && o <= OptionComeAt21OrLater
}

// DefaultOptions returns the default set of poll options
func DefaultOptions() []OptionKind {
	return []OptionKind{
		OptionComeAt19,
		OptionComeAt20,
		OptionComeAt21OrLater,
		OptionDecideLater,
		OptionNotComing,
	}
}
