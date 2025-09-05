package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/luis-e-ortega/chirpy/internal/auth"
	"github.com/luis-e-ortega/chirpy/internal/database"
)

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
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
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
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
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

func (cfg *apiConfig) updateUser(w http.ResponseWriter, r *http.Request) {
	token, tokenErr := auth.GetBearerToken(r.Header)
	if tokenErr != nil {
		cfg.respondWithError(w, r, http.StatusUnauthorized, "Something went wrong with retrieving token", tokenErr)
		return
	}
	userID, validationErr := auth.ValidateJWT(token, cfg.secret)
	if validationErr != nil {
		cfg.respondWithError(w, r, http.StatusUnauthorized, "Unauthorized", validationErr)
		return
	}

	type UserUpdate struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	userReq := UserUpdate{}
	err := decoder.Decode(&userReq)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong with decoding the request", err)
		return
	}
	if userReq.Email == "" || userReq.Password == "" {
		err = errors.New("empty fields for email or password")
		cfg.respondWithError(w, r, http.StatusBadRequest, "Email and password required", err)
		return
	}
	hashedPwd, err := auth.HashPassword(userReq.Password)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "Something went wrong with hashing password", err)
		return
	}

	ctx := r.Context()
	params := database.UpdateUserByIDParams{
		Email:          userReq.Email,
		HashedPassword: hashedPwd,
		ID:             userID,
	}

	type UserResponse struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}
	userRow, err := cfg.db.UpdateUserByID(ctx, params)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "Something went wrong while updating user info", err)
		return
	}
	resp := UserResponse{
		ID:          userRow.ID,
		CreatedAt:   userRow.CreatedAt,
		Email:       userRow.Email,
		IsChirpyRed: userRow.IsChirpyRed,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)

}
