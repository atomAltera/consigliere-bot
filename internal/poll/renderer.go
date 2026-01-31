package poll

import (
	"bytes"
	"embed"
	"html/template"
	"time"
)

//go:embed templates/*.html
var templates embed.FS

var resultsTmpl *template.Template
var titleTmpl *template.Template

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

// formatDateRussian formats a date in Russian locale
// Example: "понедельник, 15 января"
func formatDateRussian(t time.Time) string {
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
	"ruDate":      formatDateRussian,
	"ruDateShort": formatDateRussianShort,
}

func init() {
	var err error
	resultsTmpl, err = template.New("results.html").Funcs(templateFuncs).ParseFS(templates, "templates/results.html")
	if err != nil {
		panic(err)
	}
	titleTmpl, err = template.New("title.html").Funcs(templateFuncs).ParseFS(templates, "templates/title.html")
	if err != nil {
		panic(err)
	}
}

func RenderResults(results *Results) (string, error) {
	var buf bytes.Buffer
	if err := resultsTmpl.Execute(&buf, results); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderTitle renders the poll title for the given event date.
func RenderTitle(eventDate time.Time) (string, error) {
	var buf bytes.Buffer
	if err := titleTmpl.Execute(&buf, eventDate); err != nil {
		return "", err
	}
	return buf.String(), nil
}
