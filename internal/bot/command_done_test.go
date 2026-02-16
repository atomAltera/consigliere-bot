package bot

import "testing"

func TestParseStartTime(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		// Hour-only inputs
		{"19", "19:00", false},
		{"20", "20:00", false},
		{"21", "21:00", false},
		{"0", "0:00", false},
		{"9", "9:00", false},
		{"23", "23:00", false},

		// HH:MM inputs
		{"19:00", "19:00", false},
		{"20:00", "20:00", false},
		{"21:00", "21:00", false},
		{"21:30", "21:30", false},
		{"19:25", "19:25", false},
		{"0:00", "0:00", false},
		{"23:59", "23:59", false},

		// Invalid inputs
		{"abc", "", true},
		{"25:00", "", true},
		{"19:60", "", true},
		{"-1", "", true},
		{"24", "", true},
		{"19:abc", "", true},
		{"abc:00", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseStartTime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStartTime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseStartTime(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
