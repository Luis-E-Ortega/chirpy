package main

import (
	"net/http"
)

func main() {
	// Initialize multiplexer
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080", // Set for localhost
		Handler: mux,
	}

	// Adds handler for the root path
	mux.Handle("/", http.FileServer(http.Dir(".")))
	// Adds handler for assets
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	err := server.ListenAndServe()
	if err != nil {
		println("error starting listen and serve: ", err.Error())
		return
	}

	// To add handler for the root path
	mux.Handle("/", http.FileServer(http.Dir(".")))
}
