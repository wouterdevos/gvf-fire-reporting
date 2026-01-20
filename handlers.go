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

	var payload WebhookPayload
	jsonErr := json.Unmarshal(body, &payload)
	if jsonErr != nil {
		fmt.Printf("Error decoding JSON: %v\n", jsonErr)
		return
	}

	// Check if this is a Status Update (delivered/read) or a New Message
	if len(payload.Entry) > 0 && len(payload.Entry[0].Changes) > 0 {
		value := payload.Entry[0].Changes[0].Value

		if len(value.WebhookMessages) > 0 {
			// This is an actual incoming text message
			msg := value.WebhookMessages[0]
			fmt.Printf("\n[REPLY RECEIVED]\nFrom: %s\nMessage: %s\n", msg.From, msg.Text.Body)

			// Handle the incoming message
			err := app.handleIncomingMessage(msg)
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

func (app *App) handleIncomingMessage(msg WebhookMessage) error {
	state := app.getOrCreateState(msg.From)

	app.Mutex.Lock()
	defer app.Mutex.Unlock()

	var messagePayload any
	switch state.CurrentStep {
	case StepNone:
		state.CurrentStep = StepLocation
		messagePayload = app.getLocationRequestMessage(msg.From, "Welcome! To report a fire, please send the location.")
	case StepLocation:
		if msg.Type == "location" {
			latLon := fmt.Sprintf("%f,%f", msg.Location.Latitude, msg.Location.Longitude)
			state.Details["location"] = latLon
			state.CurrentStep = StepDone
			messagePayload = app.getTextMessage(msg.From, "Thank you, report received")
		} else {
			messagePayload = app.getLocationRequestMessage(msg.From, "Please use the 'Send location' button to report the fire.")
		}
	case StepDone:
		if msg.Text.Body == "New" {
			state.CurrentStep = StepLocation
			state.Details["location"] = ""
			messagePayload = app.getLocationRequestMessage(msg.From, "Starting new report. Please send location.")
		} else {
			messagePayload = app.getTextMessage(msg.From, "Report already submitted. Reply 'New' to start over.")
		}
	}

	return app.sendResponseMessage(messagePayload)
}

func (app *App) getOrCreateState(phoneNumber string) *ConversationState {
	app.Mutex.Lock()
	defer app.Mutex.Unlock()

	if state, exists := app.States[phoneNumber]; exists {
		return state
	}

	app.States[phoneNumber] = &ConversationState{
		CurrentStep: StepNone,
		Details:     make(map[string]string),
	}
	return app.States[phoneNumber]
}

func (app *App) getTextMessage(to string, body string) *TextMessage {
	message := TextMessage{
		BaseMessage: BaseMessage{
			MessagingProduct: "whatsapp",
			To:               to,
			Type:             "text",
		},
	}
	message.Text.Body = body

	return &message
}

func (app *App) getLocationRequestMessage(to string, text string) *LocationRequestMessage {
	message := LocationRequestMessage{
		BaseMessage: BaseMessage{
			MessagingProduct: "whatsapp",
			To:               to,
			Type:             "interactive",
		},
	}
	message.Interactive.Type = "location_request_message"
	message.Interactive.Body.Text = text
	message.Interactive.Action.Name = "send_location"

	return &message
}
