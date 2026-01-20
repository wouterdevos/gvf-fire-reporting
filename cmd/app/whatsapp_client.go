package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	API_VERSION = "v24.0"
	BASE_URL    = "https://graph.facebook.com/%s/%s/messages"
)

// Send a response message to a WhatsApp user
func (app *App) sendResponseMessage(messagePayload any) error {
	url := fmt.Sprintf(BASE_URL, API_VERSION, app.Config.PhoneNumberID)

	jsonData, err := json.Marshal(messagePayload)
	if err != nil {
		return err
	}

	// Create the request
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+app.Config.AccessToken)

	// Execute the request
	response, err := app.Client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	fmt.Println("Response sent successfully!")
	return nil
}
