package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"io"
	"net/http"
	"os"
)

const (
	API_VERSION = "v24.0"
	BASE_URL    = "https://graph.facebook.com/%s/%s/messages"
)

type App struct {
	Config *Config
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

type WhatsAppResponse struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body string `json:"body"`
	} `json:"text"`
}

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

func main() {
	config := loadConfig()

	app := &App{
		Config: &config,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /webhook", app.verifyServer)
	mux.HandleFunc("POST /webhook", app.handleReceivedMessage)

	fmt.Printf("Server listening on port %s...\n", config.Port)
	err := http.ListenAndServe(":"+config.Port, mux)
	if err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		os.Exit(1)
	}
}

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

// Send a response message to a person
func (app *App) sendResponseMessage(to, text string) error {
	url := fmt.Sprintf(BASE_URL, API_VERSION, app.Config.PhoneNumberID)

	// Prepare the JSON payload
	responseData := WhatsAppResponse{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
	}
	responseData.Text.Body = text

	jsonData, _ := json.Marshal(responseData)

	// Create the request
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+app.Config.AccessToken)

	// Execute the request
	client := &http.Client{}
	response, err := client.Do(request)
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
