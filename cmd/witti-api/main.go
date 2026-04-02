package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/bytetwiddler/witti/internal/httpapi"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	h := httpapi.NewHandler(time.Now, time.Local)
	log.Printf("witti API listening on %s", *addr)
	if err := http.ListenAndServe(*addr, h); err != nil {
		log.Fatal(err)
	}
}
