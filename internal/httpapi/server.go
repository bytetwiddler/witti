package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/bytetwiddler/witti"
)

type searchHTTPResponse struct {
	Success bool                 `json:"success"`
	Data    witti.SearchResponse `json:"data,omitempty"`
	Error   string               `json:"error,omitempty"`
}

// NewHandler builds the REST API handler for witti search features.
func NewHandler(now func() time.Time, local *time.Location) http.Handler {
	if now == nil {
		now = time.Now
	}
	if local == nil {
		local = time.Local
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/v1/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		var req witti.SearchRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid JSON request")
			return
		}

		resp, err := witti.Search(req, now, local)
		if err != nil {
			code := http.StatusInternalServerError
			if errors.Is(err, witti.ErrInvalidRequest) {
				code = http.StatusBadRequest
			}
			writeJSONError(w, code, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(searchHTTPResponse{Success: true, Data: resp})
	})

	return mux
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(searchHTTPResponse{Success: false, Error: message})
}
