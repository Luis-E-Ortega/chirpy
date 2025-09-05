package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/luis-e-ortega/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	secret         string
}

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

type Chirp struct {
	Body   string `json:"body"`
	UserId string `json:"user_id"`
}

type ChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    string    `json:"user_id"`
}

func main() {
	godotenv.Load() // Load env file into environment variables
	dbURL := os.Getenv("DB_URL")
	secretStr := os.Getenv("SECRET")
	//Open sql database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Error opening database connection: %s", err)
	}
	// Use sqlc generated databse package to create new *databse.Queries and store it in apiConfig struct
	// so that it can be accessed
	dbQueries := database.New(db)
	_ = dbQueries

	// Initialize multiplexer
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080", // Set for localhost
		Handler: mux,
	}

	platform := os.Getenv("PLATFORM")
	// Initialize instance of apiConfig struct
	apiCfg := apiConfig{
		db:       dbQueries,
		platform: platform,
		secret:   secretStr,
	}

	// Handle requests to /healthz with the readiness check, only allowing GET requests (others return 405)
	mux.HandleFunc("GET /api/healthz", handleHealthz)
	// Serve static files from /app/ path
	mux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
	// Endpoint to view metrics
	mux.HandleFunc("GET /admin/metrics/", apiCfg.writeHits)
	// Endpoint to reset metrics
	mux.HandleFunc("POST /admin/reset", apiCfg.reset)
	// Endpoint to allow users to create an account
	mux.HandleFunc("POST /api/users", apiCfg.createUser)
	// Endpoint to update user password
	mux.HandleFunc("PUT /api/users", apiCfg.updateUser)
	// Endpoint to allow users to post chirps to api
	mux.HandleFunc("POST /api/chirps", apiCfg.postChirp)
	// Endpoint to retrieve all chirps from a given user
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirps)
	// Endpoint to retrieve a single chirp
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirp)
	// Endpoint to delete a single chirp
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.deleteChirp)
	// Endpoint to allow user login
	mux.HandleFunc("POST /api/login", apiCfg.login)
	// Endpoint to generate refresh tokens
	mux.HandleFunc("POST /api/refresh", apiCfg.refresh)
	// Endpoint to revoke refresh tokens
	mux.HandleFunc("POST /api/revoke", apiCfg.revoke)
	// Endpoint to manage polka webhooks
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.polkaWebhooks)

	// Starts server "listening" to accept and handle requests
	fmt.Println("Server starting on port 8080...")

	err = server.ListenAndServe()
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
