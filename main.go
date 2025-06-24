package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

type Chirp struct {
	Body string `json:"body"`
}

var cfg apiConfig

var filter []string = []string{"kerfuffle", "sharbert", "fornax"}

func main() {
	mux := http.NewServeMux()

	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir("app")))))
	mux.HandleFunc("GET /api/healthz", ready)
	mux.HandleFunc("GET /admin/metrics", getMetrics)
	mux.HandleFunc("POST /admin/reset", resetMetrics)
	mux.HandleFunc("POST /api/validate_chirp", validate)

	server := http.Server{}
	server.Addr = ":8080"
	server.Handler = mux
	server.ListenAndServe()
}

func profanityFilter(s string, separator string, filter_words []string) string {
	words := strings.Split(s, separator)
	for i, word := range words {
		for _, bad := range filter_words {
			if bad == strings.ToLower(word) {
				words[i] = "****"
			}
		}
	}
	return strings.Join(words, separator)
}

func validate(w http.ResponseWriter, r *http.Request) {
	params := new(Chirp)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Error decoding parameter from JSON: %s", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `
		{
			"error": "Something went wrong"
		}
		`)
		return
	}
	if len(params.Body) > 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `
		{
			"error": "Chirp is too long"
		}
		`)
		return
	}

	reply := fmt.Sprintf(`
	{
		"cleaned_body": "%s"
	}
	`, profanityFilter(params.Body, " ", filter))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, reply)
}

func getMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	responseHTML := fmt.Sprintf(`
<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>
`, cfg.fileserverHits.Load())
	io.WriteString(w, responseHTML)
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
