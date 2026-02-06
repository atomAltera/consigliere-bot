package bot

import "nuclight.org/consigliere/internal/poll"

// Member represents a user to be mentioned in templates.
// At least one field should be set.
type Member struct {
	TgID       int64  // Telegram user ID (optional, 0 if not set)
	TgName     string // Telegram first name (optional)
	TgUsername string // Telegram username without @ (optional)
	Nickname   string // Custom nickname (optional)
}

// DisplayName returns the best available display name for the member.
// Priority: Nickname > TgName > @username (prefers game nick for display)
func (m Member) DisplayName() string {
	if m.Nickname != "" {
		return m.Nickname
	}
	if m.TgName != "" {
		return m.TgName
	}
	if m.TgUsername != "" {
		return "@" + m.TgUsername
	}
	return ""
}

// MentionName returns the name suitable for @mentions (telegram identity).
// Priority: @username > TgName (needs telegram handle for clickable mentions)
func (m Member) MentionName() string {
	if m.TgUsername != "" {
		return "@" + m.TgUsername
	}
	return m.TgName
}

// MemberFromVote creates a Member from a Vote.
func MemberFromVote(v *poll.Vote) Member {
	return Member{
		TgID:       v.TgUserID,
		TgName:     v.TgFirstName,
		TgUsername: v.TgUsername,
	}
}

// MembersFromVotes converts a slice of Votes to Members.
func MembersFromVotes(votes []*poll.Vote) []Member {
	members := make([]Member, 0, len(votes))
	for _, v := range votes {
		members = append(members, MemberFromVote(v))
	}
	return members
}
