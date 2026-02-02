package bot

import (
	"bytes"
	"embed"
	"html/template"
	"time"

	"nuclight.org/consigliere/internal/poll"
)

//go:embed templates/*
var templates embed.FS

var invitationTmpl *template.Template
var pollTitleTmpl *template.Template
var cancelTmpl *template.Template

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
	day := t.Day()
	month := russianMonths[t.Month()-1]
	return weekday + ", " + string(rune('0'+day/10)) + string(rune('0'+day%10)) + " " + month
}

// formatDateRussianShort formats a date in short Russian format
// Example: "15 января"
func formatDateRussianShort(t time.Time) string {
	day := t.Day()
	month := russianMonths[t.Month()-1]
	if day < 10 {
		return string(rune('0'+day)) + " " + month
	}
	return string(rune('0'+day/10)) + string(rune('0'+day%10)) + " " + month
}

var templateFuncs = template.FuncMap{
	"ruDate":      FormatDateRussian,
	"ruDateShort": formatDateRussianShort,
}

func init() {
	var err error
	invitationTmpl, err = template.New("invitation.html").Funcs(templateFuncs).ParseFS(templates, "templates/invitation.html")
	if err != nil {
		panic(err)
	}
	pollTitleTmpl, err = template.New("poll_title.txt").Funcs(templateFuncs).ParseFS(templates, "templates/poll_title.txt")
	if err != nil {
		panic(err)
	}
	cancelTmpl, err = template.New("cancel.txt").Funcs(templateFuncs).ParseFS(templates, "templates/cancel.txt")
	if err != nil {
		panic(err)
	}
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
func RenderInvitation(data *poll.InvitationData) (string, error) {
	var buf bytes.Buffer
	if err := invitationTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CancelData holds data for the cancel message template
type CancelData struct {
	EventDate time.Time
	Mentions  string // comma-separated @mentions
}

// RenderCancelMessage renders the cancellation notification message.
func RenderCancelMessage(data *CancelData) (string, error) {
	var buf bytes.Buffer
	if err := cancelTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// HelpMessage returns the help message HTML.
func HelpMessage() string {
	content, _ := templates.ReadFile("templates/help.html")
	return string(content)
}
