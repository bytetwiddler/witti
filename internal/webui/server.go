// Package webui serves the Witti single-page web application.
// It embeds the static index.html and handles the HTMX search endpoint
// (/ui/search) that returns HTML fragments consumed by HTMX.
package webui

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/bytetwiddler/witti"
)

//go:embed web
var webFiles embed.FS

// resultsData is the view-model rendered into the HTMX fragment.
type resultsData struct {
	Error         string
	QuerySummary  string
	ProjectedTime bool
	ReferenceTime string
	LocalZone     string
	NoMatches     bool
	OffsetMode    []string
	Results       []witti.SearchMatch
}

var resultsTmpl = template.Must(
	template.New("results").
		Funcs(template.FuncMap{
			// utcOffset converts an offset in seconds to a human-readable UTC label.
			"utcOffset": func(sec int) string {
				if sec == 0 {
					return "UTC±0"
				}
				sign := "+"
				if sec < 0 {
					sign = "−"
					sec = -sec
				}
				h := sec / 3600
				m := (sec % 3600) / 60
				if m != 0 {
					return fmt.Sprintf("UTC%s%d:%02d", sign, h, m)
				}
				return fmt.Sprintf("UTC%s%d", sign, h)
			},
		}).
		Parse(resultsHTML),
)

// NewHandler returns an http.Handler that serves the web UI at "/" and the
// HTMX search endpoint at "/ui/search".  The caller is responsible for
// mounting the JSON REST API at "/v1/" and "/healthz".
func NewHandler(now func() time.Time, local *time.Location) http.Handler {
	if now == nil {
		now = time.Now
	}
	if local == nil {
		local = time.Local
	}

	mux := http.NewServeMux()

	// ── Static SPA ──────────────────────────────────────────────────────────
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := webFiles.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})

	// ── HTMX search fragment ─────────────────────────────────────────────────
	mux.HandleFunc("/ui/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			renderFragment(w, resultsData{Error: "could not parse form"})
			return
		}

		queryStr := strings.TrimSpace(r.FormValue("query"))
		terms := tokenizeQuery(queryStr)
		localtime := strings.TrimSpace(r.FormValue("localtime"))
		localzone := strings.TrimSpace(r.FormValue("localzone"))
		use12hour := r.FormValue("use12hour") == "on"
		limit, _ := strconv.Atoi(r.FormValue("limit"))

		// Resolve the effective local location (for projection context).
		effectiveLoc := local
		if localzone != "" {
			if l, err := time.LoadLocation(localzone); err == nil {
				effectiveLoc = l
			}
		}

		req := witti.SearchRequest{
			QueryTerms:         terms,
			Use12Hour:          use12hour,
			Limit:              limit,
			ProjectedLocalTime: localtime,
			LocalTimeZone:      localzone,
		}

		resp, err := witti.Search(req, now, effectiveLoc)
		if err != nil {
			renderFragment(w, resultsData{Error: sanitizeError(err.Error())})
			return
		}

		refTimeFmt := ""
		if resp.ProjectedTime {
			refTimeFmt = resp.ReferenceTime.Format("Mon 2006-01-02 15:04:05 MST")
		}

		renderFragment(w, resultsData{
			QuerySummary:  resp.QuerySummary,
			ProjectedTime: resp.ProjectedTime,
			ReferenceTime: refTimeFmt,
			LocalZone:     localzone,
			NoMatches:     resp.NoMatches,
			OffsetMode:    resp.OffsetMode,
			Results:       resp.Results,
		})
	})

	return mux
}

// renderFragment executes the results template and writes it to w.
func renderFragment(w http.ResponseWriter, data resultsData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := resultsTmpl.Execute(w, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}

// sanitizeError strips the "invalid request: " prefix that the library adds
// so the UI shows a shorter, friendlier message.
func sanitizeError(msg string) string {
	msg = strings.TrimPrefix(msg, "invalid request: ")
	if len(msg) > 0 {
		msg = strings.ToUpper(msg[:1]) + msg[1:]
	}
	return msg
}

// tokenizeQuery splits a query string into terms, respecting double-quoted groups.
// e.g.  `los angeles "new york" tokyo`  → ["los", "angeles", "new york", "tokyo"]
// A quoted group is treated as a single term (quotes stripped).
func tokenizeQuery(s string) []string {
	var tokens []string
	var inQuote bool
	var cur strings.Builder
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
		case unicode.IsSpace(r) && !inQuote:
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// ── HTMX result fragment template ───────────────────────────────────────────

const resultsHTML = `
{{- if .Error -}}
<div class="flex items-start gap-3 p-5 rounded-2xl bg-red-50 dark:bg-red-950/60 border border-red-100 dark:border-red-900/60">
  <svg class="w-5 h-5 text-red-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
      d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"/>
  </svg>
  <p class="text-sm text-red-700 dark:text-red-300 leading-relaxed">{{.Error}}</p>
</div>
{{- else -}}

{{- if .ProjectedTime }}
<div class="flex items-start gap-3 p-4 rounded-2xl bg-blue-50 dark:bg-blue-950/50 border border-blue-100 dark:border-blue-900/50 text-sm">
  <svg class="w-4 h-4 text-blue-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/>
  </svg>
  <span class="text-blue-700 dark:text-blue-300">
    Projecting <strong class="font-semibold">{{.ReferenceTime}}</strong>{{if .LocalZone}} &mdash; from <em>{{.LocalZone}}</em>{{end}}
  </span>
</div>
{{- end}}

{{- if .OffsetMode }}
<div class="flex items-start gap-3 p-4 rounded-2xl bg-violet-50 dark:bg-violet-950/50 border border-violet-100 dark:border-violet-900/50 text-sm">
  <svg class="w-4 h-4 text-violet-500 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
      d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
  </svg>
  <span class="text-violet-700 dark:text-violet-300">
    Offset-aware mode:
    {{range $i,$v := .OffsetMode}}{{if $i}} &middot; {{end}}<strong>{{$v}}</strong>{{end}}
  </span>
</div>
{{- end}}

{{- if .NoMatches }}
<div class="text-center py-20">
  <div class="w-16 h-16 mx-auto mb-5 rounded-full bg-gray-100 dark:bg-gray-800 flex items-center justify-center">
    <svg class="w-8 h-8 text-gray-400 dark:text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
        d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
    </svg>
  </div>
  <p class="text-base font-medium text-gray-900 dark:text-white">No results found</p>
  <p class="text-sm text-gray-400 dark:text-gray-500 mt-1">No timezone entries match &#8220;{{.QuerySummary}}&#8221;</p>
</div>
{{- else}}
{{range .Results}}
<div class="group p-6 rounded-3xl bg-gray-50 dark:bg-gray-900 border border-gray-100 dark:border-gray-800
            hover:bg-white dark:hover:bg-gray-800
            hover:shadow-xl hover:shadow-gray-200/50 dark:hover:shadow-black/30
            hover:-translate-y-0.5
            transition-all duration-200 cursor-default">
  <div class="flex items-start justify-between gap-4">
    <!-- Left: time -->
    <div class="flex-1 min-w-0">
      <p class="text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider mb-2 truncate">
        {{.ZoneName}}
      </p>
      <p class="text-2xl sm:text-3xl font-semibold text-gray-900 dark:text-white tracking-tight leading-snug break-all">
        {{.FormattedTime}}
      </p>
    </div>
    <!-- Right: badge + offset -->
    <div class="flex-shrink-0 text-right">
      <span class="inline-block px-3 py-1 rounded-full text-xs font-semibold
                   bg-gray-200/70 dark:bg-gray-700 text-gray-600 dark:text-gray-300">
        {{.Abbreviation}}
      </span>
      <p class="text-xs text-gray-400 dark:text-gray-500 mt-2 font-mono">
        {{utcOffset .UTCOffsetSeconds}}
      </p>
    </div>
  </div>
</div>
{{end}}
{{- end}}
{{- end}}
`

