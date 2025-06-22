package main

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

var cfg apiConfig

func main() {
	mux := http.NewServeMux()

	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("app")))))
	mux.HandleFunc("GET /api/healthz", ready)
	mux.HandleFunc("GET /api/metrics", getMetrics)
	mux.HandleFunc("POST /api/reset", resetMetrics)

	server := http.Server{}
	server.Addr = ":8080"
	server.Handler = mux
	server.ListenAndServe()
}

func getMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load()))
}

func resetMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	cfg.fileserverHits.Store(0)
	io.WriteString(w, "Metrics Reset")
}

func ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
