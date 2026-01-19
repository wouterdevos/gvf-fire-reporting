package main

import "net/http"

type App struct {
	Config *Config
	Client *http.Client
}

type Config struct {
	VerifyToken   string
	AccessToken   string
	PhoneNumberID string
	Port          string
}

// Payload received from the WhatsApp webhooks. Includes incoming messages and any updates to messages sent
// to people including delivered and read messages
type WhatsAppPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					From string `json:"from"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
				Statuses []interface{} `json:"statuses"` // To detect status updates
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// A WhatsApp text message payload sent via WhatsApp Cloud API
type WhatsAppResponse struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body string `json:"body"`
	} `json:"text"`
}
