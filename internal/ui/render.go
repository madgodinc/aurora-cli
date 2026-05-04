package ui

import (
	"strings"

	"charm.land/glamour/v2"
)

var mdRenderer *glamour.TermRenderer

func init() {
	var err error
	mdRenderer, err = glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		mdRenderer = nil
	}
}

// renderMarkdown renders markdown text to styled terminal output.
func renderMarkdown(text string) string {
	if mdRenderer == nil || strings.TrimSpace(text) == "" {
		return text
	}
	out, err := mdRenderer.Render(text)
	if err != nil {
		return text
	}
	return strings.TrimRight(out, "\n")
}
