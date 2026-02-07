package bot

import (
	"testing"
)

func TestParseNickArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     *NickArgs
		wantErr  bool
		errMatch string
	}{
		{
			name:  "simple username and nick",
			input: "@user1 Секртис",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Секртис",
				Gender:     GenderNotSet,
			},
		},
		{
			name:  "numeric ID and nick",
			input: "123456789 Кринж",
			want: &NickArgs{
				TgUserID: int64Ptr(123456789),
				Nickname: "Кринж",
				Gender:   GenderNotSet,
			},
		},
		{
			name:  "username with quoted nick containing spaces",
			input: `@user1 "Мадам Жу"`,
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Мадам Жу",
				Gender:     GenderNotSet,
			},
		},
		{
			name:  "username with single quoted nick",
			input: `@user1 'Мадам Жу'`,
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Мадам Жу",
				Gender:     GenderNotSet,
			},
		},
		{
			name:  "username with guillemets",
			input: "@user1 «Мадам Жу»",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Мадам Жу",
				Gender:     GenderNotSet,
			},
		},
		{
			name:  "username with smart quotes",
			input: "@user1 \u201cМадам Жу\u201d",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Мадам Жу",
				Gender:     GenderNotSet,
			},
		},
		{
			name:  "with male gender (m)",
			input: "@user1 Кринж m",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Кринж",
				Gender:     GenderMale,
			},
		},
		{
			name:  "with male gender (м)",
			input: "@user1 Кринж м",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Кринж",
				Gender:     GenderMale,
			},
		},
		{
			name:  "with female gender (f)",
			input: "@user1 Каэтана f",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Каэтана",
				Gender:     GenderFemale,
			},
		},
		{
			name:  "with female gender (ж)",
			input: "@user1 Каэтана ж",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Каэтана",
				Gender:     GenderFemale,
			},
		},
		{
			name:  "with female gender (д)",
			input: "@user1 Каэтана д",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Каэтана",
				Gender:     GenderFemale,
			},
		},
		{
			name:  "quoted nick with gender",
			input: `@user1 "Мадам Жу" ж`,
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Мадам Жу",
				Gender:     GenderFemale,
			},
		},
		{
			name:  "ID with quoted nick and gender",
			input: `12345 "Мадам Жу" f`,
			want: &NickArgs{
				TgUserID: int64Ptr(12345),
				Nickname: "Мадам Жу",
				Gender:   GenderFemale,
			},
		},
		{
			name:  "uppercase gender",
			input: "@user1 Кринж M",
			want: &NickArgs{
				TgUsername: strPtr("user1"),
				Nickname:   "Кринж",
				Gender:     GenderMale,
			},
		},
		// Error cases
		{
			name:     "empty input",
			input:    "",
			wantErr:  true,
			errMatch: "not enough arguments",
		},
		{
			name:     "only identifier",
			input:    "@user1",
			wantErr:  true,
			errMatch: "not enough arguments",
		},
		{
			name:     "empty username",
			input:    "@ Nick",
			wantErr:  true,
			errMatch: "empty username",
		},
		{
			name:     "invalid identifier",
			input:    "notausername Nick",
			wantErr:  true,
			errMatch: "invalid identifier",
		},
		{
			name:     "negative ID",
			input:    "-123 Nick",
			wantErr:  true,
			errMatch: "invalid identifier",
		},
		{
			name:     "invalid gender",
			input:    "@user1 Nick x",
			wantErr:  true,
			errMatch: "invalid gender",
		},
		{
			name:     "too many arguments",
			input:    "@user1 Nick m extra",
			wantErr:  true,
			errMatch: "too many arguments",
		},
		{
			name:     "unclosed quote",
			input:    `@user1 "Мадам Жу`,
			wantErr:  true,
			errMatch: "unclosed quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseNickArgs(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseNickArgs() expected error containing %q, got nil", tt.errMatch)
					return
				}
				if tt.errMatch != "" && !contains(err.Error(), tt.errMatch) {
					t.Errorf("ParseNickArgs() error = %q, want error containing %q", err.Error(), tt.errMatch)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseNickArgs() unexpected error: %v", err)
				return
			}

			if !nickArgsEqual(got, tt.want) {
				t.Errorf("ParseNickArgs() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGenderString(t *testing.T) {
	tests := []struct {
		gender Gender
		want   string
	}{
		{GenderNotSet, ""},
		{GenderMale, "male"},
		{GenderFemale, "female"},
	}

	for _, tt := range tests {
		if got := tt.gender.String(); got != tt.want {
			t.Errorf("Gender(%d).String() = %q, want %q", tt.gender, got, tt.want)
		}
	}
}

func TestGenderFromString(t *testing.T) {
	tests := []struct {
		input string
		want  Gender
	}{
		{"", GenderNotSet},
		{"male", GenderMale},
		{"female", GenderFemale},
		{"unknown", GenderNotSet},
	}

	for _, tt := range tests {
		if got := GenderFromString(tt.input); got != tt.want {
			t.Errorf("GenderFromString(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestGenderPrefix(t *testing.T) {
	tests := []struct {
		gender Gender
		want   string
	}{
		{GenderNotSet, ""},
		{GenderMale, "г-н"},
		{GenderFemale, "г-ж"},
	}

	for _, tt := range tests {
		if got := tt.gender.Prefix(); got != tt.want {
			t.Errorf("Gender(%d).Prefix() = %q, want %q", tt.gender, got, tt.want)
		}
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func int64Ptr(i int64) *int64 {
	return &i
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func nickArgsEqual(a, b *NickArgs) bool {
	if a == nil || b == nil {
		return a == b
	}

	// Compare TgUserID
	if (a.TgUserID == nil) != (b.TgUserID == nil) {
		return false
	}
	if a.TgUserID != nil && *a.TgUserID != *b.TgUserID {
		return false
	}

	// Compare TgUsername
	if (a.TgUsername == nil) != (b.TgUsername == nil) {
		return false
	}
	if a.TgUsername != nil && *a.TgUsername != *b.TgUsername {
		return false
	}

	return a.Nickname == b.Nickname && a.Gender == b.Gender
}
