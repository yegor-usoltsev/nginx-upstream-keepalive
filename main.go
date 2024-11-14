package main

import (
	"log"
	"net/http"
)

const KeepAlivesEnabled = true // Enabled by default

func main() {
	srv := &http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(handler),
	}

	srv.SetKeepAlivesEnabled(KeepAlivesEnabled)

	log.Printf("Starting HTTP server on port 8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %s", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request from %s | Protocol: %s | Will be closed: %t", r.RemoteAddr, r.Proto, r.Close || !KeepAlivesEnabled)
	w.Write([]byte("Hello, World\n"))
}
