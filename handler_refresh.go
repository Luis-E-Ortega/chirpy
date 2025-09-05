package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/luis-e-ortega/chirpy/internal/auth"
)

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
