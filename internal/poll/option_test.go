package poll

import "testing"

func TestOptionKind_Label(t *testing.T) {
	tests := []struct {
		opt  OptionKind
		want string
	}{
		{OptionComeAt19, "Приду к 19:00"},
		{OptionComeAt20, "Приду к 20:00"},
		{OptionComeAt21OrLater, "Приду к 21:00 или позже"},
		{OptionDecideLater, "Решу позже"},
		{OptionNotComing, "Не приду"},
	}
	for _, tt := range tests {
		if got := tt.opt.Label(); got != tt.want {
			t.Errorf("OptionKind(%d).Label() = %q, want %q", tt.opt, got, tt.want)
		}
	}
}

func TestOptionKind_IsAttending(t *testing.T) {
	attending := []OptionKind{OptionComeAt19, OptionComeAt20, OptionComeAt21OrLater}
	notAttending := []OptionKind{OptionDecideLater, OptionNotComing}

	for _, opt := range attending {
		if !opt.IsAttending() {
			t.Errorf("OptionKind(%d).IsAttending() = false, want true", opt)
		}
	}
	for _, opt := range notAttending {
		if opt.IsAttending() {
			t.Errorf("OptionKind(%d).IsAttending() = true, want false", opt)
		}
	}
}

func TestAllOptions(t *testing.T) {
	opts := AllOptions()
	if len(opts) != 5 {
		t.Errorf("AllOptions() returned %d options, want 5", len(opts))
	}
	if opts[0] != "Приду к 19:00" {
		t.Errorf("AllOptions()[0] = %q, want %q", opts[0], "Приду к 19:00")
	}
}
