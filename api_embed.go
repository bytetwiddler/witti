package witti

import _ "embed"

// apiGuideMarkdown is the canonical Markdown API guide used by the web UI docs page.
//
//go:embed api.md
var apiGuideMarkdown string

// APIGuideMarkdown returns the embedded Markdown API guide.
func APIGuideMarkdown() string {
	return apiGuideMarkdown
}

