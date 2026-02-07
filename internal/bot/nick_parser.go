package bot

import (
	"errors"
	"strconv"
	"strings"
)

// Gender represents the gender of a player.
type Gender int

const (
	GenderNotSet Gender = iota
	GenderMale
	GenderFemale
)

// String returns the string representation for database storage.
func (g Gender) String() string {
	switch g {
	case GenderMale:
		return "male"
	case GenderFemale:
		return "female"
	default:
		return ""
	}
}

// GenderFromString parses a gender string from database.
func GenderFromString(s string) Gender {
	switch s {
	case "male":
		return GenderMale
	case "female":
		return GenderFemale
	default:
		return GenderNotSet
	}
}

// Prefix returns the display prefix for the gender.
func (g Gender) Prefix() string {
	switch g {
	case GenderMale:
		return "г-н"
	case GenderFemale:
		return "г-ж"
	default:
		return ""
	}
}

// NickArgs holds parsed /nick command arguments.
type NickArgs struct {
	TgUserID   *int64  // Telegram user ID (set if identifier is numeric)
	TgUsername *string // Telegram username without @ (set if identifier starts with @)
	Nickname   string  // Game nickname
	Gender     Gender  // Gender (optional)
}

// ParseNickArgs parses /nick command arguments with shell-style quoting.
// Supports: /nick @user nick, /nick @user "nick with spaces", /nick @user nick m
func ParseNickArgs(input string) (*NickArgs, error) {
	tokens, err := tokenize(input)
	if err != nil {
		return nil, err
	}

	if len(tokens) < 2 {
		return nil, errors.New("not enough arguments")
	}

	result := &NickArgs{}

	// Parse identifier (first token)
	identifier := tokens[0]
	if strings.HasPrefix(identifier, "@") {
		username := strings.TrimPrefix(identifier, "@")
		if username == "" {
			return nil, errors.New("empty username")
		}
		result.TgUsername = &username
	} else if id, err := strconv.ParseInt(identifier, 10, 64); err == nil && id > 0 {
		result.TgUserID = &id
	} else {
		return nil, errors.New("invalid identifier: must be @username or numeric ID")
	}

	// Parse nickname (second token)
	result.Nickname = tokens[1]
	if result.Nickname == "" {
		return nil, errors.New("empty nickname")
	}

	// Parse optional gender (third token)
	if len(tokens) >= 3 {
		gender, err := parseGender(tokens[2])
		if err != nil {
			return nil, err
		}
		result.Gender = gender
	}

	// Too many arguments
	if len(tokens) > 3 {
		return nil, errors.New("too many arguments")
	}

	return result, nil
}

// parseGender parses a gender marker.
// Valid values: m, f, м, д, ж (case-insensitive)
func parseGender(s string) (Gender, error) {
	switch strings.ToLower(s) {
	case "m", "м":
		return GenderMale, nil
	case "f", "д", "ж":
		return GenderFemale, nil
	default:
		return GenderNotSet, errors.New("invalid gender: use m/f/м/д/ж")
	}
}

// tokenize splits input into tokens, respecting quoted strings.
// Supports both single and double quotes.
func tokenize(input string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	var inQuote rune
	escaped := false

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if inQuote != 0 {
			if r == inQuote {
				inQuote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}

		// Handle various quote characters
		// " (U+0022) - standard double quote
		// ' (U+0027) - standard single quote
		// « (U+00AB) - left guillemet, closes with » (U+00BB)
		// " (U+201C) - left double quotation mark, closes with " (U+201D)
		switch r {
		case '"', '\'':
			inQuote = r
			continue
		case '«':
			inQuote = '»'
			continue
		case '\u201c': // "
			inQuote = '\u201d' // "
			continue
		}

		if r == ' ' || r == '\t' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(r)
	}

	if inQuote != 0 {
		return nil, errors.New("unclosed quote")
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}
