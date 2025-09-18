package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/luis-e-ortega/chirpy/internal/auth"
	"github.com/luis-e-ortega/chirpy/internal/database"
)

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
	s := r.URL.Query().Get("author_id")

	var authorNullUUID uuid.NullUUID

	if s == "" {
		// If s is empty, create a NullUUID that represents SQL NULL
		authorNullUUID = uuid.NullUUID{Valid: false}
	} else {
		parsedID, err := uuid.Parse(s)
		if err != nil {
			cfg.respondWithError(w, r, 404, "error parsing string into uuid", err)
			return
		}
		authorNullUUID = uuid.NullUUID{UUID: parsedID, Valid: true}

	}

	chirps, err := cfg.db.GetAllChirps(r.Context(), authorNullUUID)
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

	// Incase a sort parameter is present in the query, use it to sort chirps (sorting is already defaulted to asc)
	sortParam := strings.ToLower(r.URL.Query().Get("sort"))
	if sortParam == "desc" {
		sort.Slice(formattedResp, func(i, j int) bool {
			return formattedResp[i].CreatedAt.After(formattedResp[j].CreatedAt)
		})
	} else {
		sort.Slice(formattedResp, func(i, j int) bool {
			return formattedResp[i].CreatedAt.Before(formattedResp[j].CreatedAt)
		})
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

func (cfg *apiConfig) deleteChirp(w http.ResponseWriter, r *http.Request) {
	chirpID := r.PathValue("chirpID")

	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	verifiedID, err := auth.ValidateJWT(bearerToken, cfg.secret)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusUnauthorized, "unauthorized", err)
		return
	}

	parsedID, err := uuid.Parse(chirpID)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusNotFound, "Error parsing string into UUID.", err)
		return
	}
	chirp, err := cfg.db.GetSingleChirp(r.Context(), parsedID)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusNotFound, "Error retrieving chirp.", err)
		return
	}
	if !chirp.UserID.Valid {
		cfg.respondWithError(w, r, http.StatusForbidden, "forbidden", errors.New("chirp has no author"))
		return
	}

	if chirp.UserID.UUID != verifiedID {
		cfg.respondWithError(w, r, http.StatusForbidden, "forbidden", errors.New("not the author"))
		return
	}

	// All checks passed, delete the chirp

	if err := cfg.db.DeleteChirpByID(r.Context(), parsedID); err != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "failed to delete chirp", err)
		return
	}

	// Header for successful delete
	w.WriteHeader(http.StatusNoContent)
}
