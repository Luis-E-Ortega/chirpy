package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	// Initialize multiplexer
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080", // Set for localhost
		Handler: mux,
	}

	// Initialize instance of apiConfig struct
	apiCfg := apiConfig{}

	// Handle requests to /healthz with the readiness check, only allowing GET requests (others return 405)
	mux.HandleFunc("GET /healthz", handleHealthz)
	// Serve static files from /app/ path
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	// Serve metrics from /metrics path, only allowing GET requests
	mux.HandleFunc("GET /metrics/", apiCfg.writeHits)
	// Serve reset from /reset path, only allowing POST requests
	mux.HandleFunc("POST /reset", apiCfg.reset)

	err := server.ListenAndServe()
	if err != nil {
		println("error starting listen and serve: ", err.Error())
		return
	}
}

func handleHealthz(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

// Increments and keeps track of count per request to server
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1) // Need to use this syntax to safely add to our atomic int
		next.ServeHTTP(w, r)
	})
}

// Writes the hits on the site to the response
func (cfg *apiConfig) writeHits(w http.ResponseWriter, req *http.Request) {
	str := fmt.Sprintf("Hits: %v", cfg.fileserverHits.Load())
	w.Write([]byte(str))
}

// Resets hits count to 0
func (cfg *apiConfig) reset(w http.ResponseWriter, req *http.Request) {
	cfg.fileserverHits.Swap(0)
}
