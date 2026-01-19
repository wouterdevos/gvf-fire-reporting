package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (app *App) verifyServer(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received verification request")

	query := r.URL.Query()
	mode := query.Get("hub.mode")
	token := query.Get("hub.verify_token")
	challenge := query.Get("hub.challenge")

	if mode != "subscribe" || token != app.Config.VerifyToken {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	fmt.Println("Verification successful!")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(challenge))
}

// Handle a message received via the WhatsApp webhook
func (app *App) handleReceivedMessage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received message")
	body, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		fmt.Printf("Error reading body: %v\n", readErr)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// This will show you the "Raw" WhatsApp JSON structure in your terminal
	fmt.Printf("Received WhatsApp JSON: %s\n", string(body))

	var payload WhatsAppPayload
	jsonErr := json.Unmarshal(body, &payload)
	if jsonErr != nil {
		fmt.Printf("Error decoding JSON: %v\n", jsonErr)
		return
	}

	// Check if this is a Status Update (delivered/read) or a New Message
	if len(payload.Entry) > 0 && len(payload.Entry[0].Changes) > 0 {
		value := payload.Entry[0].Changes[0].Value

		if len(value.Messages) > 0 {
			// This is an actual incoming text message
			msg := value.Messages[0]
			fmt.Printf("\n[REPLY RECEIVED]\nFrom: %s\nMessage: %s\n", msg.From, msg.Text.Body)

			// Trigger the replay
			responseText := "You said: " + msg.Text.Body
			err := app.sendResponseMessage(msg.From, responseText)
			if err != nil {
				fmt.Printf("Failed to send reply: %v\n", err)
			}
		} else if len(value.Statuses) > 0 {
			// This is just Meta telling you a message was "delivered" or "read"
			fmt.Println("Status update received (delivered/read).")
		}
	}

	w.WriteHeader(http.StatusOK)
}
