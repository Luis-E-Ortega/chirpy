package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/luis-e-ortega/chirpy/internal/auth"
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
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
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
	// Serve metrics from /metrics path, only allowing GET requests
	mux.HandleFunc("GET /admin/metrics/", apiCfg.writeHits)
	// Handle reset from /reset path, only allowing POST requests
	mux.HandleFunc("POST /admin/reset", apiCfg.reset)
	// Handle users to be able to create them
	mux.HandleFunc("POST /api/users", apiCfg.createUser)
	// Handle chirps endpoint to allow users to post chirps to api
	mux.HandleFunc("POST /api/chirps", apiCfg.postChirp)
	// Serve chirps (return all) from /api/chirps path
	mux.HandleFunc("GET /api/chirps", apiCfg.getChirps)
	// Serve chirp (returning one) from /api/chirp/{chirpID} path
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirp)
	// Serve login from /api/login path
	mux.HandleFunc("POST /api/login", apiCfg.login)
	// Serve refresh from /api/refresh path
	mux.HandleFunc("POST /api/refresh", apiCfg.refresh)
	// Serve revoke from /api/revoke path
	mux.HandleFunc("POST /api/revoke", apiCfg.revoke)

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

/*
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
}*/

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
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with decoding the request.", err)
		return
	}

	hashedPwd, err := auth.HashPassword(userReq.Password)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "Something went wrong with hashing the password.", err)
		return
	}

	userParams := database.CreateUserParams{
		Email:          userReq.Email,
		HashedPassword: hashedPwd,
	}
	user, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with creating the user.", err)
		return
	}

	userData := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	dat, err := json.Marshal(userData)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "Something went wrong with marshalling the request.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(dat)
}

func (cfg *apiConfig) postChirp(w http.ResponseWriter, r *http.Request) {
	token, tokenErr := auth.GetBearerToken(r.Header)
	if tokenErr != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with retrieving token", tokenErr)
		return
	}
	userID, validationErr := auth.ValidateJWT(token, cfg.secret)
	if validationErr != nil {
		cfg.respondWithError(w, r, 401, "Unauthorized", validationErr)
		return
	}
	decoder := json.NewDecoder(r.Body)
	chirp := Chirp{}
	err := decoder.Decode(&chirp)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with decoding the request.", err)
		return
	}

	if len(chirp.Body) > 140 {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Chirp is too long.", nil)
		return
	} else {

		params := database.CreateChirpParams{
			Body:   profaneReplacer(chirp.Body),
			UserID: uuid.NullUUID{UUID: userID, Valid: true},
		}
		chirp, err := cfg.db.CreateChirp(r.Context(), params)
		if err != nil {
			cfg.respondWithError(w, r, http.StatusInternalServerError, "Error creating chirp.", err)
			return
		}

		resp := ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.UUID.String(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(resp)
	}
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.db.GetAllChirps(r.Context())
	if err != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "Error retrieving chirps from database.", err)
		return
	}

	formattedResp := []ChirpResponse{}

	for _, chirp := range chirps {
		resp := ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    chirp.UserID.UUID.String(),
		}
		formattedResp = append(formattedResp, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(formattedResp)

}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("chirpID")
	parsedID, err := uuid.Parse(chirpID)
	if err != nil {
		cfg.respondWithError(w, r, 404, "Error parsing string into UUID.", err)
		return
	}

	chirp, err := cfg.db.GetSingleChirp(r.Context(), parsedID)
	if err != nil {
		cfg.respondWithError(w, r, 404, "Error retrieving chirp.", err)
		return
	}

	chirpResp := ChirpResponse{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    parsedID.String(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(chirpResp)
}

func (cfg *apiConfig) login(w http.ResponseWriter, r *http.Request) {
	// Creating login-specific structs
	type loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type loginResponse struct {
		User
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	decoder := json.NewDecoder(r.Body)
	userReq := loginRequest{}
	err := decoder.Decode(&userReq)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with decoding the request.", err)
		return
	}
	userEmail := userReq.Email
	user, err := cfg.db.GetUser(r.Context(), userEmail)
	if err != nil {
		cfg.respondWithError(w, r, 401, "Incorrect email or password", err)
		return
	}
	err = auth.CheckPasswordHash(userReq.Password, user.HashedPassword)
	if err != nil {
		cfg.respondWithError(w, r, 401, "Incorrect email or password", err)
		return
	}

	formattedExpires := time.Duration(3600) * time.Second
	token, err := auth.MakeJWT(user.ID, cfg.secret, formattedExpires)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with authorizing the token.", err)
		return
	}

	userResponse := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}

	rawRefreshToken, TokenErr := auth.MakeRefreshToken()
	if TokenErr != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "Something went wrong with making the refresh token.", TokenErr)
		return
	}
	expireTime := time.Now().Add(time.Duration(time.Hour * 1440))
	params := database.CreateRefreshTokenParams{
		Token: rawRefreshToken,
		ExpiresAt: sql.NullTime{
			Time:  expireTime,
			Valid: true,
		},
		UserID: uuid.NullUUID{
			UUID:  user.ID,
			Valid: true,
		},
	}
	refreshToken, err := cfg.db.CreateRefreshToken(r.Context(), params)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "Something went wrong with creating the refresh token.", err)
		return
	}
	response := loginResponse{
		User:         userResponse,
		Token:        token,
		RefreshToken: refreshToken.Token,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(response)
}

func (cfg *apiConfig) refresh(w http.ResponseWriter, r *http.Request) {
	type loginResponse struct {
		Token string `json:"token"`
	}

	bearerToken, tokenErr := auth.GetBearerToken(r.Header)
	if tokenErr != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with retrieving token", tokenErr)
		return
	}
	refreshToken, err := cfg.db.GetRefreshToken(r.Context(), bearerToken)
	if err != nil {
		cfg.respondWithError(w, r, 401, "Mismatched or non-existing token", err)
		return
	}

	formattedExpires := time.Duration(3600) * time.Second
	token, err := auth.MakeJWT(refreshToken.UserID.UUID, cfg.secret, formattedExpires)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with authorizing the token.", err)
		return
	}

	response := loginResponse{
		Token: token,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(response)
}

func (cfg *apiConfig) revoke(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with retrieving token", err)
		return
	}
	refreshToken, err := cfg.db.GetRefreshToken(r.Context(), bearerToken)
	if err != nil {
		cfg.respondWithError(w, r, 401, "Mismatched or non-existing token", err)
		return
	}

	revokeErr := cfg.db.RevokeRefreshToken(r.Context(), refreshToken.Token)
	if revokeErr != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with revoking the refresh token", revokeErr)
		return
	}
	w.WriteHeader(204)
}
