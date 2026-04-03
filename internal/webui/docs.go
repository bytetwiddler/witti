package webui

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/bytetwiddler/witti/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

var apiGuideRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(
		gmhtml.WithHardWraps(),
	),
)

var apiGuidePageTmpl = template.Must(template.New("api-guide-page").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Witti API Guide</title>
  <style>
    :root {
      color-scheme: light dark;
      --bg: #ffffff;
      --fg: #1f2328;
      --muted: #59636e;
      --border: #d0d7de;
      --surface: #f6f8fa;
      --surface-2: #f3f4f6;
      --link: #0969da;
      --code-bg: rgba(175,184,193,0.2);
    }
    @media (prefers-color-scheme: dark) {
      :root {
        --bg: #0d1117;
        --fg: #e6edf3;
        --muted: #9da7b3;
        --border: #30363d;
        --surface: #161b22;
        --surface-2: #151b23;
        --link: #58a6ff;
        --code-bg: rgba(110,118,129,0.4);
      }
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      background: var(--bg);
      color: var(--fg);
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      line-height: 1.5;
    }
    .shell {
      max-width: 980px;
      margin: 0 auto;
      padding: 24px;
    }
    .topbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      gap: 16px;
      margin-bottom: 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid var(--border);
    }
    .brand {
      font-size: 14px;
      font-weight: 600;
      letter-spacing: -0.01em;
    }
    .nav {
      display: flex;
      gap: 16px;
      flex-wrap: wrap;
    }
    .nav a {
      color: var(--muted);
      text-decoration: none;
      font-size: 14px;
    }
    .nav a:hover { color: var(--link); }
    .markdown-body {
      font-size: 16px;
      line-height: 1.65;
      word-wrap: break-word;
    }
    .markdown-body h1,
    .markdown-body h2,
    .markdown-body h3,
    .markdown-body h4,
    .markdown-body h5,
    .markdown-body h6 {
      margin-top: 24px;
      margin-bottom: 16px;
      font-weight: 600;
      line-height: 1.25;
      letter-spacing: -0.02em;
    }
    .markdown-body h1 {
      margin-top: 0;
      padding-bottom: 0.3em;
      font-size: 2em;
      border-bottom: 1px solid var(--border);
    }
    .markdown-body h2 {
      padding-bottom: 0.3em;
      font-size: 1.5em;
      border-bottom: 1px solid var(--border);
    }
    .markdown-body p,
    .markdown-body ul,
    .markdown-body ol,
    .markdown-body table,
    .markdown-body blockquote,
    .markdown-body pre {
      margin-top: 0;
      margin-bottom: 16px;
    }
    .markdown-body a {
      color: var(--link);
      text-decoration: none;
    }
    .markdown-body a:hover { text-decoration: underline; }
    .markdown-body code {
      padding: 0.2em 0.4em;
      margin: 0;
      font-size: 85%;
      background-color: var(--code-bg);
      border-radius: 6px;
      font-family: ui-monospace, SFMono-Regular, SFMono-Regular, Menlo, Consolas, monospace;
    }
    .markdown-body pre {
      padding: 16px;
      overflow: auto;
      font-size: 85%;
      line-height: 1.45;
      background-color: var(--surface);
      border-radius: 6px;
    }
    .markdown-body pre code {
      padding: 0;
      background: transparent;
      border-radius: 0;
    }
    .markdown-body blockquote {
      margin-left: 0;
      padding: 0 1em;
      color: var(--muted);
      border-left: 0.25em solid var(--border);
    }
    .markdown-body table {
      display: block;
      width: max-content;
      max-width: 100%;
      overflow: auto;
      border-spacing: 0;
      border-collapse: collapse;
    }
    .markdown-body table th,
    .markdown-body table td {
      padding: 6px 13px;
      border: 1px solid var(--border);
    }
    .markdown-body table tr {
      background-color: var(--bg);
      border-top: 1px solid var(--border);
    }
    .markdown-body table tr:nth-child(2n) {
      background-color: var(--surface-2);
    }
    .footer {
      margin-top: 32px;
      padding-top: 16px;
      border-top: 1px solid var(--border);
      color: var(--muted);
      font-size: 14px;
    }
  </style>
</head>
<body>
  <div class="shell">
    <div class="topbar">
      <div class="brand">Witti API Guide</div>
      <div class="nav">
        <a href="/">Home</a>
        <a href="/api.md">Raw Markdown</a>
        <a href="https://github.com/bytetwiddler/witti" target="_blank" rel="noopener">GitHub</a>
      </div>
    </div>
    <article class="markdown-body">{{.Body}}</article>
    <div class="footer">Rendered from the embedded <code>api.md</code> guide using GitHub-flavored Markdown.</div>
  </div>
</body>
</html>
`))

func renderAPIGuidePage(w http.ResponseWriter) {
	var body bytes.Buffer
	if err := apiGuideRenderer.Convert([]byte(witti.APIGuideMarkdown()), &body); err != nil {
		http.Error(w, "could not render API guide", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := apiGuidePageTmpl.Execute(w, struct{ Body template.HTML }{Body: template.HTML(body.String())}); err != nil {
		http.Error(w, "could not render API guide page", http.StatusInternalServerError)
	}
}
