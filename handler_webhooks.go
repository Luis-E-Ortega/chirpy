package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
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
