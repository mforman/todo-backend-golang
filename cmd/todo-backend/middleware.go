package main

import (
	"log"
	"net/http"
	"time"
)

func optionsOk(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("access-control-allow-origin", "*")
		w.Header().Set("access-control-allow-methods", "GET, POST, PATCH, DELETE")
		w.Header().Set("access-control-allow-headers", "accept, content-type")
		if r.Method == "OPTIONS" {
			return // Preflight sets headers and we're done
		}
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func contentTypeJsonHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func commonHandlers(next http.HandlerFunc) http.Handler {
	return loggingHandler(contentTypeJsonHandler(optionsOk(next)))
}
