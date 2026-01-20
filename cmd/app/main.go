package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	config := loadConfig()

	app := &App{
		Config: &config,
		Client: &http.Client{},
		States: make(map[string]*ConversationState),
	}

	if err := app.Run(); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		os.Exit(1)
	}
}
