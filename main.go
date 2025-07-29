package main

import (
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

	// Handle requests to /healthz with the readiness check
	mux.HandleFunc("/healthz", handleHealthz)
	// Serve static files from /app/ path
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))

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

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
