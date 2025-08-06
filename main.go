package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	mux.HandleFunc("GET /api/healthz", handleHealthz)
	// Serve static files from /app/ path
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	// Serve metrics from /metrics path, only allowing GET requests
	mux.HandleFunc("GET /admin/metrics/", apiCfg.writeHits)
	// Handle reset from /reset path, only allowing POST requests
	mux.HandleFunc("POST /admin/reset", apiCfg.reset)
	// Handle chirp validation
	mux.HandleFunc("POST /api/validate_chirp", apiCfg.validate)

	// Starts server "listening" to accept and handle requests
	fmt.Println("Server starting on port 8080...")

	err := server.ListenAndServe()
	if err != nil {
		println("error starting listen and serve: ", err.Error())
		return
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
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
	cfg.fileserverHits.Swap(0)
}

func (cfg *apiConfig) validate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters : %s", err)
		w.WriteHeader(500)
		return
	}

	if len(params.Body) > 140 {
		// Creating a struct with only error field to return
		errorResp := struct {
			Error string `json:"error"`
		}{
			Error: "Chirp is too long",
		}
		w.WriteHeader(400)
		dat, err := json.Marshal(errorResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(dat)
	} else {
		// Creating a struct with only the valid field to return
		cleanedResp := struct {
			CleanedBody string `json:"cleaned_body"`
		}{
			CleanedBody: profaneReplacer(params.Body),
		}
		w.WriteHeader(200)
		dat, err := json.Marshal(cleanedResp)
		if err != nil {
			log.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(dat)
	}
}

func profaneReplacer(s string) string {
	bannedWords := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	words := strings.Split(s, " ")

	for i, word := range words {
		for _, profaneWord := range bannedWords {
			if strings.ToLower(word) == profaneWord {
				words[i] = "****"
			}
		}
	}
	joinedstr := strings.Join(words, " ")
	return joinedstr
}
