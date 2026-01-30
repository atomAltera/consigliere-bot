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

func init() {
	var err error
	resultsTmpl, err = template.ParseFS(templates, "templates/results.html")
	if err != nil {
		panic(err)
	}
	titleTmpl, err = template.ParseFS(templates, "templates/title.html")
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
