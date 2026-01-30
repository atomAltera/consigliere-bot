package poll

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templates embed.FS

var resultsTmpl *template.Template

func init() {
	var err error
	resultsTmpl, err = template.ParseFS(templates, "templates/results.html")
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
