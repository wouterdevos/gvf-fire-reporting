package main

import (
	"fmt"
	"net/http"
)

func (app *App) Run() error {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /webhook", app.verifyServer)
	mux.HandleFunc("POST /webhook", app.handleReceivedMessage)

	fmt.Printf("Server listening on port %s...\n", app.Config.Port)
	return http.ListenAndServe(":"+app.Config.Port, mux)
}
