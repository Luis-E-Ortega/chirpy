package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/luis-e-ortega/chirpy/internal/auth"
)

type WebhooksEvent struct {
	Event string   `json:"event"`
	Data  UserData `json:"data"`
}

type UserData struct {
	UserID string `json:"user_id"`
}

func (cfg *apiConfig) polkaWebhooks(w http.ResponseWriter, r *http.Request) {
	evt := WebhooksEvent{}
	err := json.NewDecoder(r.Body).Decode(&evt)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "error decoding request", err)
		return
	}

	actualAPIKey := cfg.polkaKey
	requestAPIKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		cfg.respondWithError(w, r, 401, "error getting api key", err)
		return
	}

	if requestAPIKey != actualAPIKey {
		keyErr := errors.New("invalid api key")
		cfg.respondWithError(w, r, http.StatusBadRequest, "unauthorized request", keyErr)
		return
	}

	if evt.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	id, err := uuid.Parse(evt.Data.UserID)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusBadRequest, "invalid user_id", err)
		return
	}

	rows, err := cfg.db.UpgradeUserToRed(r.Context(), id)
	if err != nil {
		cfg.respondWithError(w, r, http.StatusInternalServerError, "upgrade failed", err)
		return
	}

	if rows == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}
