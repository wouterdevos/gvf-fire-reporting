package main

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
)

func loadConfig() Config {
	// Load the .env file for dev (ignore error if file doesn't exist for prod)
	_ = godotenv.Load()

	config := Config{
		VerifyToken:   os.Getenv("VERIFY_TOKEN"),
		AccessToken:   os.Getenv("ACCESS_TOKEN"),
		PhoneNumberID: os.Getenv("PHONE_NUMBER_ID"),
		Port:          os.Getenv("PORT"),
	}

	if config.VerifyToken == "" || config.AccessToken == "" || config.PhoneNumberID == "" {
		fmt.Println("CRICICAL: Missing required environment variables!")
		os.Exit(1)
	}

	if config.Port == "" {
		// Default value
		config.Port = "8080"
	}

	return config
}
