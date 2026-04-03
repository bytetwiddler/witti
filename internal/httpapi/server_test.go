package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytetwiddler/witti"
)

func TestSearchEndpoint(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC) }
	la, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	h := NewHandler(now, la)

	body, _ := json.Marshal(witti.SearchRequest{QueryTerms: []string{"02/17/2027 07:07:00", "new york"}, Limit: 1})
	req := httptest.NewRequest(http.MethodPost, "/v1/search", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Success bool                 `json:"success"`
		Data    witti.SearchResponse `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Success {
		t.Fatalf("expected success=true body=%s", rec.Body.String())
	}
	if !payload.Data.ProjectedTime {
		t.Fatalf("expected projected time flag")
	}
	if len(payload.Data.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(payload.Data.Results))
	}
	if payload.Data.Results[0].ZoneName != "America/New_York" {
		t.Fatalf("unexpected zone: %s", payload.Data.Results[0].ZoneName)
	}
	r0 := payload.Data.Results[0]
	if r0.GMTLabel == "" {
		t.Fatalf("expected GMTLabel to be populated, got empty string")
	}
	if r0.UTCTime == "" {
		t.Fatalf("expected UTCTime to be populated, got empty string")
	}
	// Feb 2027: NYC=EST=UTC-5; compact label must be "GMT-5"
	if r0.GMTLabel != "GMT-5" {
		t.Fatalf("expected GMTLabel=GMT-5, got %q", r0.GMTLabel)
	}
	// UTC representation of the projected instant
	if !strings.Contains(r0.UTCTime, "UTC") {
		t.Fatalf("expected UTC in UTCTime, got %q", r0.UTCTime)
	}
}

func TestSearchEndpointBadRequest(t *testing.T) {
	h := NewHandler(time.Now, time.Local)

	req := httptest.NewRequest(http.MethodPost, "/v1/search", bytes.NewBufferString(`{"queryTerms":["02/17/2027 07:07"]}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSearchEndpointMethodNotAllowed(t *testing.T) {
	h := NewHandler(time.Now, time.Local)

	req := httptest.NewRequest(http.MethodGet, "/v1/search", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "method not allowed") {
		t.Fatalf("expected method-not-allowed message, got: %s", rec.Body.String())
	}
}

func TestSearchEndpointUnknownField(t *testing.T) {
	h := NewHandler(time.Now, time.Local)

	req := httptest.NewRequest(http.MethodPost, "/v1/search", bytes.NewBufferString(`{"queryTerms":["tokyo"],"unknown":1}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "invalid JSON request") {
		t.Fatalf("expected invalid-json message, got: %s", rec.Body.String())
	}
}

func TestSearchEndpointInvalidRequestSemantic(t *testing.T) {
	h := NewHandler(time.Now, time.Local)

	req := httptest.NewRequest(http.MethodPost, "/v1/search", bytes.NewBufferString(`{"queryTerms":["tokyo"],"localTimeZone":"Not/AZone"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHealthz(t *testing.T) {
	h := NewHandler(time.Now, time.Local)

	t.Run("get ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})
}
