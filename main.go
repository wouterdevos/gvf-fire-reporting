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
	}

	if err := app.Run(); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		os.Exit(1)
	}
}
