package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/bytetwiddler/witti/internal/httpapi"
	"github.com/bytetwiddler/witti/internal/webui"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	// Combined mux: REST API + web UI
	mux := http.NewServeMux()

	// JSON REST API (served at /v1/ and /healthz)
	apiHandler := httpapi.NewHandler(time.Now, time.Local)
	mux.Handle("/v1/", apiHandler)
	mux.Handle("/healthz", apiHandler)

	// Web UI (SPA at /, HTMX fragment at /ui/search)
	webHandler := webui.NewHandler(time.Now, time.Local)
	mux.Handle("/", webHandler)

	log.Printf("witti web  →  http://localhost%s", *addr)
	log.Printf("witti API  →  http://localhost%s/v1/search", *addr)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

