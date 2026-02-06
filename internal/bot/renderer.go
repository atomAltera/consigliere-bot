package bot

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

// TelegramMaxMessageLength is the maximum length of a Telegram message (4096 chars)
const TelegramMaxMessageLength = 4096

//go:embed templates/*
var templates embed.FS

var invitationTmpl *template.Template
var pollTitleTmpl *template.Template
var cancelTmpl *template.Template
var restoreTmpl *template.Template
var callTmpl *template.Template
var collectedTmpl *template.Template
var resultsTmpl *template.Template

// Russian weekday names
var russianWeekdays = []string{
	"воскресенье",
	"понедельник",
	"вторник",
	"среда",
	"четверг",
	"пятница",
	"суббота",
}

// Russian month names in genitive case (for dates like "15 января")
var russianMonths = []string{
	"января",
	"февраля",
	"марта",
	"апреля",
	"мая",
	"июня",
	"июля",
	"августа",
	"сентября",
	"октября",
	"ноября",
	"декабря",
}

// FormatDateRussian formats a date in Russian locale
// Example: "понедельник, 15 января"
func FormatDateRussian(t time.Time) string {
	weekday := russianWeekdays[t.Weekday()]
	month := russianMonths[t.Month()-1]
	return fmt.Sprintf("%s, %d %s", weekday, t.Day(), month)
}

// formatDateRussianShort formats a date in short Russian format
// Example: "15 января"
func formatDateRussianShort(t time.Time) string {
	month := russianMonths[t.Month()-1]
	return fmt.Sprintf("%d %s", t.Day(), month)
}

// formatMembers formats a slice of Members as space-separated display names
func formatMembers(members []Member) string {
	if len(members) == 0 {
		return ""
	}
	names := make([]string, 0, len(members))
	for _, m := range members {
		if name := m.DisplayName(); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, " ")
}

// formatMentions formats a slice of Members as space-separated @mentions
func formatMentions(members []Member) string {
	if len(members) == 0 {
		return ""
	}
	names := make([]string, 0, len(members))
	for _, m := range members {
		if name := m.MentionName(); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, " ")
}

// formatCollectedMembers formats members for the /done message.
// Shows numbered list with @username and nickname on each line.
func formatCollectedMembers(members []Member) string {
	if len(members) == 0 {
		return ""
	}
	lines := make([]string, 0, len(members))
	for i, m := range members {
		var line string
		if m.TgUsername != "" {
			line = "@" + m.TgUsername
			if m.Nickname != "" {
				line += " " + m.Nickname
			}
		} else if m.Nickname != "" {
			line = m.Nickname
		} else if m.TgName != "" {
			line = m.TgName
		}
		if line != "" {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, line))
		}
	}
	return strings.Join(lines, "\n")
}

// formatNickList formats members as comma-separated nicks (or names if no nick).
func formatNickList(members []Member) string {
	if len(members) == 0 {
		return ""
	}
	names := make([]string, 0, len(members))
	for _, m := range members {
		var name string
		if m.Nickname != "" {
			name = m.Nickname
		} else if m.TgName != "" {
			name = m.TgName
		} else if m.TgUsername != "" {
			name = "@" + m.TgUsername
		}
		if name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

var templateFuncs = template.FuncMap{
	"ruDate":                 FormatDateRussian,
	"ruDateShort":            formatDateRussianShort,
	"formatMembers":          formatMembers,
	"formatMentions":         formatMentions,
	"formatCollectedMembers": formatCollectedMembers,
	"formatNickList":         formatNickList,
}

// InitTemplates initializes all templates. Must be called before using any Render* functions.
func InitTemplates() error {
	var err error
	invitationTmpl, err = template.New("invitation.html").Funcs(templateFuncs).ParseFS(templates, "templates/invitation.html")
	if err != nil {
		return fmt.Errorf("parse invitation template: %w", err)
	}
	pollTitleTmpl, err = template.New("poll_title.txt").Funcs(templateFuncs).ParseFS(templates, "templates/poll_title.txt")
	if err != nil {
		return fmt.Errorf("parse poll title template: %w", err)
	}
	cancelTmpl, err = template.New("cancel.html").Funcs(templateFuncs).ParseFS(templates, "templates/cancel.html")
	if err != nil {
		return fmt.Errorf("parse cancel template: %w", err)
	}
	restoreTmpl, err = template.New("restore.html").Funcs(templateFuncs).ParseFS(templates, "templates/restore.html")
	if err != nil {
		return fmt.Errorf("parse restore template: %w", err)
	}
	callTmpl, err = template.New("call.html").Funcs(templateFuncs).ParseFS(templates, "templates/call.html")
	if err != nil {
		return fmt.Errorf("parse call template: %w", err)
	}
	collectedTmpl, err = template.New("collected.html").Funcs(templateFuncs).ParseFS(templates, "templates/collected.html")
	if err != nil {
		return fmt.Errorf("parse collected template: %w", err)
	}
	resultsTmpl, err = template.New("results.html").Funcs(templateFuncs).ParseFS(templates, "templates/results.html")
	if err != nil {
		return fmt.Errorf("parse results template: %w", err)
	}
	return nil
}

// RenderPollTitle renders the poll title for the given event date.
func RenderPollTitle(eventDate time.Time) (string, error) {
	var buf bytes.Buffer
	if err := pollTitleTmpl.Execute(&buf, eventDate); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderInvitation renders the invitation message for the given data.
// If the message exceeds Telegram's limit, participant lists are truncated.
func RenderInvitation(data *poll.InvitationData) (string, error) {
	var buf bytes.Buffer
	if err := invitationTmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	result := buf.String()
	if len(result) <= TelegramMaxMessageLength {
		return result, nil
	}

	// Message too long - truncate participant lists progressively
	// Make a copy to avoid modifying the original data
	truncatedData := &poll.InvitationData{
		Poll:        data.Poll,
		EventDate:   data.EventDate,
		IsCancelled: data.IsCancelled,
	}

	// Copy slices
	truncatedData.Participants = append([]*poll.Vote{}, data.Participants...)
	truncatedData.ComingLater = append([]*poll.Vote{}, data.ComingLater...)
	truncatedData.Undecided = append([]*poll.Vote{}, data.Undecided...)

	// Truncate until message fits, removing from the longest list first
	for len(result) > TelegramMaxMessageLength {
		// Find the longest list and remove one item
		maxLen := 0
		var longest *[]*poll.Vote
		if len(truncatedData.Participants) > maxLen {
			maxLen = len(truncatedData.Participants)
			longest = &truncatedData.Participants
		}
		if len(truncatedData.ComingLater) > maxLen {
			maxLen = len(truncatedData.ComingLater)
			longest = &truncatedData.ComingLater
		}
		if len(truncatedData.Undecided) > maxLen {
			longest = &truncatedData.Undecided
		}

		if longest == nil || len(*longest) == 0 {
			// Nothing left to truncate, return what we have
			break
		}

		// Remove the last item from the longest list
		*longest = (*longest)[:len(*longest)-1]

		// Re-render
		buf.Reset()
		if err := invitationTmpl.Execute(&buf, truncatedData); err != nil {
			return "", err
		}
		result = buf.String()
	}

	return result, nil
}

// CancelData holds data for the cancel message template
type CancelData struct {
	EventDate time.Time
	Members   []Member
}

// RenderCancelMessage renders the cancellation notification message.
func RenderCancelMessage(data *CancelData) (string, error) {
	var buf bytes.Buffer
	if err := cancelTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RestoreData holds data for the restore message template
type RestoreData struct {
	EventDate time.Time
	Members   []Member
}

// RenderRestoreMessage renders the restore notification message.
func RenderRestoreMessage(data *RestoreData) (string, error) {
	var buf bytes.Buffer
	if err := restoreTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CallData holds data for the call message template
type CallData struct {
	EventDate time.Time
	Members   []Member
}

// RenderCallMessage renders the call message for undecided voters.
func RenderCallMessage(data *CallData) (string, error) {
	var buf bytes.Buffer
	if err := callTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// HelpMessage returns the help message HTML.
func HelpMessage() (string, error) {
	content, err := templates.ReadFile("templates/help.html")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// CollectedData holds data for the collected message template
type CollectedData struct {
	EventDate   time.Time
	StartTime   string // e.g., "19:00" or "20:00"
	Members     []Member
	ComingLater []Member // Players coming at 21:00+
}

// RenderCollectedMessage renders the "players collected" notification message.
func RenderCollectedMessage(data *CollectedData) (string, error) {
	var buf bytes.Buffer
	if err := collectedTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ResultsVoter holds voter info for the /results admin display
type ResultsVoter struct {
	TgID       int64
	TgUsername string
	TgName     string
	Nickname   string
}

// ResultsData holds data for the results admin message template
type ResultsData struct {
	EventDate   time.Time
	At19        []ResultsVoter
	At20        []ResultsVoter
	ComingLater []ResultsVoter
	Undecided   []ResultsVoter
}

// RenderResultsMessage renders the admin results message.
func RenderResultsMessage(data *ResultsData) (string, error) {
	var buf bytes.Buffer
	if err := resultsTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
