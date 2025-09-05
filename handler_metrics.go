package main

import (
	"fmt"
	"net/http"
)

// Increments and keeps track of count per request to server
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1) // Need to use this syntax to safely add to our atomic int
		next.ServeHTTP(w, r)
	})
}

// Writes the hits on the site to the response
func (cfg *apiConfig) writeHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	str := fmt.Sprintf(`
	<html>
	<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
	</body>
	</html>`, cfg.fileserverHits.Load())
	w.Write([]byte(str))
}

// Resets hits count to 0
func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		return
	}
	cfg.fileserverHits.Swap(0)
	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		fmt.Printf("Error deleting users: %s", err)
	}
}
