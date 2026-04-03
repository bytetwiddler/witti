package webui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// fixedNow returns a deterministic time for tests.
func fixedNow() time.Time {
	return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load America/Los_Angeles: %v", err)
	}
	return NewHandler(fixedNow, la)
}

// ── GET / ─────────────────────────────────────────────────────────────────────

func TestIndexHTML(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status=%d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("unexpected Content-Type: %s", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "What Is The Time In") {
		t.Fatal("index.html missing hero text")
	}
	if !strings.Contains(body, `href="/api"`) {
		t.Fatal("index.html missing API docs link")
	}
}

func TestIndexNotFound(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/does/not/exist", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestAPIGuidePage(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("unexpected Content-Type: %s", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Witti API Guide") {
		t.Fatalf("expected API guide title: %s", body)
	}
	if !strings.Contains(body, "POST /v1/search") {
		t.Fatalf("expected search endpoint docs: %s", body)
	}
	if !strings.Contains(body, "Rendered from the embedded") {
		t.Fatalf("expected footer note: %s", body)
	}
}

func TestAPIGuideRawMarkdown(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api.md", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api.md status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/markdown") {
		t.Fatalf("unexpected Content-Type: %s", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "# witti REST API") {
		t.Fatalf("expected markdown heading: %s", body)
	}
	if !strings.Contains(body, "## `POST /v1/search`") {
		t.Fatalf("expected markdown endpoint section: %s", body)
	}
}

func TestAPIGuideMethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)

	for _, path := range []string{"/api", "/api.md"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("POST %s: expected 405, got %d", path, rec.Code)
		}
	}
}

// ── POST /ui/search ──────────────────────────────────────────────────────────

func postSearch(t *testing.T, h http.Handler, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/ui/search", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSearchBasic(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{"query": []string{"new york"}, "limit": []string{"1"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "New_York") {
		t.Fatalf("expected New_York in body, got: %s", body)
	}
	if !strings.Contains(body, "GMT-4") {
		t.Fatalf("expected GMT detail line in body, got: %s", body)
	}
	if !strings.Contains(body, "12:00:00 UTC") {
		t.Fatalf("expected UTC detail line in body, got: %s", body)
	}
	if !strings.Contains(body, "text-sm font-medium text-gray-500") {
		t.Fatalf("expected muted UTC styling in body, got: %s", body)
	}
}

func TestSearchMultipleTerms(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{"query": []string{"los angeles new york"}, "limit": []string{"2"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Los_Angeles") {
		t.Fatalf("expected Los_Angeles in body: %s", body)
	}
	if !strings.Contains(body, "New_York") {
		t.Fatalf("expected New_York in body: %s", body)
	}
}

func TestSearchProjectedTime(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{
		"query":     []string{"new york"},
		"localtime": []string{"02/17/2027 07:07:00"},
		"localzone": []string{"America/Los_Angeles"},
		"limit":     []string{"1"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Projecting") {
		t.Fatalf("expected projection banner: %s", body)
	}
	// New York is UTC-5 in February; LA is UTC-8 → +3h → 10:07:00
	if !strings.Contains(body, "10:07") {
		t.Fatalf("expected 10:07 in projected result: %s", body)
	}
	if !strings.Contains(body, "GMT-5") {
		t.Fatalf("expected projected GMT detail line: %s", body)
	}
	if !strings.Contains(body, "15:07:00 UTC") {
		t.Fatalf("expected projected UTC detail line: %s", body)
	}
}

func TestSearch12HourClock(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{
		"query":     []string{"new york"},
		"limit":     []string{"1"},
		"use12hour": []string{"on"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	// 12-hour format includes AM or PM
	if !strings.Contains(body, "AM") && !strings.Contains(body, "PM") {
		t.Fatalf("expected AM/PM in 12-hour result: %s", body)
	}
	// GMT label is appended inline on the local time line
	if !strings.Contains(body, "(GMT-4)") {
		t.Fatalf("expected (GMT-4) inline label in 12-hour result: %s", body)
	}
	if !strings.Contains(body, "12:00:00 PM UTC") {
		t.Fatalf("expected UTC detail line to honor 12-hour format: %s", body)
	}
}

func TestSearchGMTOffset(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{"query": []string{"gmt-8"}, "limit": []string{"5"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	// Should contain offset mode banner
	if !strings.Contains(body, "Offset-aware") {
		t.Fatalf("expected offset-aware banner for gmt-8: %s", body)
	}
}

func TestSearchNoMatches(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{"query": []string{"zzznomatchzzz"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "No results found") {
		t.Fatalf("expected no-results message: %s", rec.Body.String())
	}
}

func TestSearchInvalidProjectedTime(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{
		"query":     []string{"tokyo"},
		"localtime": []string{"not-a-date"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	// Error fragment should be rendered (not HTTP error)
	body := rec.Body.String()
	if !strings.Contains(body, "M12 9v2") { // SVG path in error banner
		// Accept any error rendering
		if !strings.Contains(body, "bg-red") {
			t.Fatalf("expected error fragment for bad localtime: %s", body)
		}
	}
}

func TestSearchMethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/ui/search", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

// ── tokenizeQuery ─────────────────────────────────────────────────────────────

func TestTokenizeQuery(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"los angeles", []string{"los", "angeles"}},
		{`"los angeles"`, []string{"los angeles"}},
		{`"los angeles" "new york"`, []string{"los angeles", "new york"}},
		{"tokyo", []string{"tokyo"}},
		{"  ", nil},
		{"", nil},
		{`gmt-8 "buenos aires"`, []string{"gmt-8", "buenos aires"}},
	}
	for _, tc := range cases {
		got := tokenizeQuery(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("tokenizeQuery(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("tokenizeQuery(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

// ── sanitizeError ─────────────────────────────────────────────────────────────

func TestSanitizeError(t *testing.T) {
	cases := []struct{ in, want string }{
		{"invalid request: foo bar", "Foo bar"},
		{"some other error", "Some other error"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := sanitizeError(tc.in); got != tc.want {
			t.Errorf("sanitizeError(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// ── NewHandler nil guards ────────────────────────────────────────────────────

func TestNewHandlerNilGuards(t *testing.T) {
	// Should not panic with nil arguments
	h := NewHandler(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("nil-guard handler: status=%d", rec.Code)
	}
}

// ── GET /ui/zones ─────────────────────────────────────────────────────────────

func TestZonesEndpoint(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/ui/zones", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /ui/zones status=%d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("unexpected Content-Type: %s", ct)
	}
	var zones []string
	if err := json.Unmarshal(rec.Body.Bytes(), &zones); err != nil {
		t.Fatalf("decode /ui/zones: %v", err)
	}
	if len(zones) == 0 {
		t.Fatal("expected non-empty zones list")
	}
	// List must be sorted
	for i := 1; i < len(zones); i++ {
		if zones[i] < zones[i-1] {
			t.Fatalf("zones not sorted: %q before %q", zones[i-1], zones[i])
		}
	}
	// Well-known zone must be present
	found := false
	for _, z := range zones {
		if z == "America/Los_Angeles" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("America/Los_Angeles missing from /ui/zones")
	}
}

func TestZonesEndpointMethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/ui/zones", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}

