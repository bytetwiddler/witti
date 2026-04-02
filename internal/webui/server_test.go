package webui

import (
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
	if !strings.Contains(rec.Body.String(), "What Is The Time In") {
		t.Fatal("index.html missing hero text")
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

	rec := postSearch(t, h, url.Values{"query": {"new york"}, "limit": {"1"}})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "New_York") {
		t.Fatalf("expected New_York in body, got: %s", body)
	}
}

func TestSearchMultipleTerms(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{"query": {"los angeles new york"}, "limit": {"2"}})

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
		"query":     {"new york"},
		"localtime": {"02/17/2027 07:07:00"},
		"localzone": {"America/Los_Angeles"},
		"limit":     {"1"},
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
}

func TestSearch12HourClock(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{
		"query":     {"london"},
		"limit":     {"1"},
		"use12hour": {"on"},
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	// 12-hour format includes AM or PM
	if !strings.Contains(body, "AM") && !strings.Contains(body, "PM") {
		t.Fatalf("expected AM/PM in 12-hour result: %s", body)
	}
}

func TestSearchGMTOffset(t *testing.T) {
	h := newTestHandler(t)

	rec := postSearch(t, h, url.Values{"query": {"gmt-8"}, "limit": {"5"}})

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

	rec := postSearch(t, h, url.Values{"query": {"zzznomatchzzz"}})

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
		"query":     {"tokyo"},
		"localtime": {"not-a-date"},
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

