package bot

import (
	_ "embed"
)

//go:embed templates/help.html
var helpTemplate string

// HelpMessage returns the help message HTML.
func HelpMessage() string {
	return helpTemplate
}
