package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
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
}

type parameters struct {
	Body string `json:"body"`
}

type CreateUserRequest struct {
	Email string `json:"email"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func main() {
	godotenv.Load() // Load env file into environment variables
	dbURL := os.Getenv("DB_URL")
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
	}

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
	// Handle users to be able to create them
	mux.HandleFunc("POST /api/users", apiCfg.createUser)

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

func (cfg *apiConfig) validate(w http.ResponseWriter, r *http.Request) {
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

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	userReq := CreateUserRequest{}
	err := decoder.Decode(&userReq)
	if err != nil {
		fmt.Printf("Error decoding user : %s", err)
	}
	// Comments for debugging
	fmt.Printf("Decoded email: '%s'\n", userReq.Email)
	user, err := cfg.db.CreateUser(r.Context(), userReq.Email)
	if err != nil {
		fmt.Printf("Error creating user: %s", err)
	}
	fmt.Printf("Database user: %+v\n", user)
	userData := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	dat, err := json.Marshal(userData)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(dat)

}
