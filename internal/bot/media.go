package bot

import (
	"embed"
	"fmt"
	"strings"
	"time"
)

//go:embed media/*/*
var mediaFS embed.FS

// GetEventVideo returns the video bytes for the given weekday from the media directory.
// Returns nil and false if mediaDir is empty or the file doesn't exist.
func GetEventVideo(mediaDir string, weekday time.Weekday) ([]byte, bool) {
	if mediaDir == "" {
		return nil, false
	}

	filename := fmt.Sprintf("media/%s/%s.mp4", mediaDir, strings.ToLower(weekday.String()))
	data, err := mediaFS.ReadFile(filename)
	if err != nil {
		return nil, false
	}
	return data, true
}
