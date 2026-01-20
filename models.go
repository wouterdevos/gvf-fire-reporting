package main

import (
	"net/http"
	"sync"
)

/*
The application running on the server
*/
type App struct {
	Config *Config
	Client *http.Client
	States map[string]*ConversationState
	Mutex  sync.RWMutex
}

/*
Configuration information required for the server and communication with WhatsApp's API
*/
type Config struct {
	VerifyToken   string
	AccessToken   string
	PhoneNumberID string
	Port          string
}

type PromptStep int

/*
The set of prompts to display to a WhatsApp user
*/
const (
	StepNone PromptStep = iota
	StepLocation
	StepDone
)

/*
The state of a conversation between a WhatsApp user and the bot
*/
type ConversationState struct {
	CurrentStep PromptStep
	Details     map[string]string
}

/*
A webhook message received via WhatsApp webhooks. The same payload is used for incoming and outgoing messages
and the main differences relate to the message Type. The different types that are currently used are:
"text" - Plain text message contained within Text struct
"location" - Location information contained within the Location struct
*/
type WebhookPayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				WebhookMessages []WebhookMessage `json:"messages"`
				Statuses        []interface{}    `json:"statuses"` // To detect status updates
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

/*
A message received within a WhatsApp webhook payload
*/
type WebhookMessage struct {
	From string `json:"from"`
	Type string `json:"type"`
	Text struct {
		Body string `json:"body"`
	} `json:"text"`
	Location WebhookLocation `json:"location"`
}

/*
A location received within a WhatsApp webhook message
*/
type WebhookLocation struct {
	Address   string  `json:"address"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Name      string  `json:"name"`
	URL       string  `json:"url"`
}

/*
The data shared among all WhatsApp message payloads sent via WhatsApp Cloud API
*/
type BaseMessage struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
}

/*
A WhatsApp text message payload sent via WhatsApp Cloud API
*/
type TextMessage struct {
	BaseMessage
	Text struct {
		Body string `json:"body"`
	} `json:"text"`
}

/*
A WhatsApp location request message payload sent via WhatsApp Cloud API
*/
type LocationRequestMessage struct {
	BaseMessage
	Interactive struct {
		Type string `json:"type"`
		Body struct {
			Text string `json:"text"`
		} `json:"body"`
		Action struct {
			Name string `json:"name"`
		} `json:"action"`
	} `json:"interactive"`
}
