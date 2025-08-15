package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (cfg *apiConfig) respondWithError(w http.ResponseWriter, r *http.Request, code int, msg string, err error) {
	fmt.Printf("Responding with error: %s for client. Internal error: %s", msg, err)
	// Anonymous struct for the JSON error response
	errorResponse := struct {
		Error string `json:"error"`
	}{
		Error: msg,
	}
	// Marshal the error response to JSON
	jsonError, marshalErr := json.Marshal(errorResponse)
	if marshalErr != nil {
		//Handle the unlikely case where marshalling the error itself fails
		fmt.Printf("Error marshalling error reponse: %s", marshalErr)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(jsonError)
}
