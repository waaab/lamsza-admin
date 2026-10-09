package main

import (
	"net/http"
	"testing"
	"time"
)

// The server must never go back to a bare http.ListenAndServe: without these
// timeouts a slowloris client holds a connection for as long as it likes.
func TestNewServerSetsEveryTimeout(t *testing.T) {
	srv := newServer(":0", http.NotFoundHandler())

	checks := map[string]time.Duration{
		"ReadHeaderTimeout": srv.ReadHeaderTimeout,
		"ReadTimeout":       srv.ReadTimeout,
		"WriteTimeout":      srv.WriteTimeout,
		"IdleTimeout":       srv.IdleTimeout,
	}
	for name, got := range checks {
		if got <= 0 {
			t.Errorf("%s is not set (%v)", name, got)
		}
	}
	if srv.MaxHeaderBytes <= 0 {
		t.Errorf("MaxHeaderBytes is not set (%d)", srv.MaxHeaderBytes)
	}
}

// The slowest request (an 8 MB upload's read (60s) plus the relay's 10s client) must finish
// before WriteTimeout cuts the connection.
func TestWriteTimeoutOutlastsTheSlowestHandler(t *testing.T) {
	const slowestHandler = 70 * time.Second

	srv := newServer(":0", http.NotFoundHandler())
	if srv.WriteTimeout <= slowestHandler {
		t.Fatalf("WriteTimeout = %v, must be longer than %v", srv.WriteTimeout, slowestHandler)
	}
}
