package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	REPLY_REPORT_ID   = "report-reply"
	REPLY_DONATE_ID   = "donate-reply"
	REPLY_CONTACTS_ID = "contacts-reply"
	REPLY_EFT_ID      = "eft-reply"
	REPLY_SNAPSCAN_ID = "snapscan-id"

	START_MENU_WELCOME = "Welcome!"
	START_MENU_INFO = "Please select an option below to proceed."
	START_MENU_HINT = "To start again reply 'Menu'."

	REPORT_INFO = "To report a fire, please send the location."
	REPORT_INFO_EXPLICIT = "Please use the 'Send location' button to report the fire."
	REPORT_RECEIVED = "Thank you, report received"

	DONATION_INFO = "Please select one of the options provided to make a donation."
	DONATION_BANKING_DETAILS = "To make an EFT please use the following banking details:\n\nFNB\nGreyton Volunteer Firefighters NPC\nAccount Number - 63131550287\nBranch Code - 200212\n\nPlease use your name and surname as reference."
	DONATION_SNAPSCAN_INFO = "To make a donation with SnapScan please follow the link provided."
	DONATION_SNAPSCAN_URL = "https://pos.snapscan.io/qr/U_F7xasA"
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
		state.CurrentStep = StepMenu
		messagePayload = app.getStartMenuMessage(msg.From, START_MENU_WELCOME+" "+START_MENU_INFO)
	case StepMenu:
		if !(msg.Type == "interactive" && msg.Interactive.Type == "button_reply") {
			messagePayload = app.getStartMenuMessage(msg.From, START_MENU_INFO)
			break
		}
		id := msg.Interactive.ButtonReply.ID
		switch id {
		case REPLY_REPORT_ID:
			state.CurrentStep = StepReport
			messagePayload = app.getLocationRequestMessage(msg.From, REPORT_INFO)
		case REPLY_DONATE_ID:
			state.CurrentStep = StepDonate
			messagePayload = app.getDonationMenuMessage(msg.From, DONATION_INFO)
		case REPLY_CONTACTS_ID:
			state.CurrentStep = StepDone
			messagePayload = app.getTextMessage(msg.From, "Here are the contacts.")
		default:
			messagePayload = app.getStartMenuMessage(msg.From, START_MENU_INFO)
		}
	case StepReport:
		if msg.Type == "location" {
			latLon := fmt.Sprintf("%f,%f", msg.Location.Latitude, msg.Location.Longitude)
			state.Details["location"] = latLon
			state.CurrentStep = StepDone
			messagePayload = app.getTextMessage(msg.From, REPORT_RECEIVED)
		} else {
			messagePayload = app.getLocationRequestMessage(msg.From, REPORT_INFO_EXPLICIT)
		}
	case StepDonate:
		if !(msg.Type == "interactive" && msg.Interactive.Type == "button_reply") {
			messagePayload = app.getDonationMenuMessage(msg.From, DONATION_INFO)
			break
		}
		id := msg.Interactive.ButtonReply.ID
		switch id {
		case REPLY_EFT_ID:
			state.CurrentStep = StepDone
			messagePayload = app.getTextMessage(msg.From, DONATION_BANKING_DETAILS)
		case REPLY_SNAPSCAN_ID:
			state.CurrentStep = StepDone
			messagePayload = app.getURLButtonMessage(msg.From, DONATION_SNAPSCAN_INFO, "SnapScan", DONATION_SNAPSCAN_URL)
		default:
			messagePayload = app.getDonationMenuMessage(msg.From, DONATION_INFO)
		}
	case StepDone:
		// Ignore the case of the text and check if "menu" was returned by the customer
		if strings.EqualFold(msg.Text.Body, "Menu") {
			state.CurrentStep = StepMenu
			state.Details["location"] = ""
			messagePayload = app.getStartMenuMessage(msg.From, START_MENU_INFO)
		} else {
			messagePayload = app.getTextMessage(msg.From, START_MENU_HINT)
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

func (app *App) getReplyButtonsMessage(to string, text string, buttons []ReplyButton) *ReplyButtonsMessage {
	message := ReplyButtonsMessage{
		BaseMessage: BaseMessage{
			MessagingProduct: "whatsapp",
			To:               to,
			Type:             "interactive",
		},
	}
	message.Interactive.Type = "button"
	message.Interactive.Body.Text = text
	message.Interactive.Action.Buttons = buttons

	return &message
}

func (app *App) getReplyButton(id string, title string) ReplyButton {
	return ReplyButton{
		Type: "reply",
		Reply: ButtonValue{
			ID:    id,
			Title: title,
		},
	}
}

func (app *App) getURLButtonMessage(to string, text string, displayText string, url string) *URLButtonMessage {
	message := URLButtonMessage{
		BaseMessage: BaseMessage{
			MessagingProduct: "whatsapp",
			To:               to,
			Type:             "interactive",
		},
	}
	message.Interactive.Type = "cta_url"
	message.Interactive.Body.Text = text
	message.Interactive.Action.Name = "cta_url"
	message.Interactive.Action.Parameters.DisplayText = displayText
	message.Interactive.Action.Parameters.URL = url

	return &message
}

func (app *App) getStartMenuMessage(to string, text string) *ReplyButtonsMessage {
	replyButtons := []ReplyButton{
		app.getReplyButton(REPLY_REPORT_ID, "Report a fire"),
		app.getReplyButton(REPLY_DONATE_ID, "Donate money"),
		app.getReplyButton(REPLY_CONTACTS_ID, "Emergency numbers"),
	}
	return app.getReplyButtonsMessage(to, text, replyButtons)
}

func (app *App) getDonationMenuMessage(to string, text string) *ReplyButtonsMessage {
	replyButtons := []ReplyButton{
		app.getReplyButton(REPLY_EFT_ID, "EFT"),
		app.getReplyButton(REPLY_SNAPSCAN_ID, "SnapScan"),
	}
	return app.getReplyButtonsMessage(to, text, replyButtons)
}
