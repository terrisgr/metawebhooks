package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type TemplateMessage struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Template         struct {
		Name     string `json:"name"`
		Language struct {
			Code string `json:"code"`
		} `json:"language"`
		Components []struct { // Add this
			Type       string `json:"type"`
			Parameters []struct {
				Type  string `json:"type"`
				Image struct {
					Link string `json:"link,omitempty"`
				} `json:"image,omitempty"`
			} `json:"parameters,omitempty"`
		} `json:"components,omitempty"`
	} `json:"template"`
}

// Webhook verification token (set in your Meta App settings)
var verifyToken = "331959e6-a3ba-891d-b3ea-d3737dceb4c20e"
var token = os.Getenv("VERIFICATION_TOKEN")

// Struct for POST webhook body
type WebhookEvent struct {
	Object string           `json:"object"`
	Entry  []map[string]any `json:"entry"`
}

// GET /webhooks - for verification
func handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("hub.mode")
	token := r.URL.Query().Get("hub.verify_token")
	challenge := r.URL.Query().Get("hub.challenge")

	if mode == "subscribe" && token == verifyToken {
		fmt.Fprintf(w, "%s", challenge) // return challenge back
		log.Println("Webhook verified successfully")
	} else {
		http.Error(w, "Forbidden", http.StatusForbidden)
	}
}

// POST /webhooks - for receiving events
func handlePostWebhook(w http.ResponseWriter, r *http.Request, cache *Cache) {
	var event WebhookEvent
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	log.Printf("Webhook event received: %+v\n", event)

	// Iterate over entries
	for _, entry := range event.Entry {
		if changes, ok := entry["changes"].([]any); ok {
			for _, ch := range changes {
				if changeMap, ok := ch.(map[string]any); ok {
					if field, ok := changeMap["field"].(string); ok && field == "messages" {
						if value, ok := changeMap["value"].(map[string]any); ok {
							var fromPhone string
							if messages, ok := value["messages"].([]any); ok {
								for _, m := range messages {
									if msg, ok := m.(map[string]any); ok {
										if from, ok := msg["from"].(string); ok {
											fromPhone = from
										}
										if msgType, ok := msg["type"].(string); ok && msgType == "button" {
											if button, ok := msg["button"].(map[string]any); ok {
												payload := button["payload"]
												text := button["text"]

												switch payload {
												case "Keybox / Keys":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "keybox", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												case "Rafina Port → Apartment":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "rafinatoairport", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												case "Wi-Fi":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "wifi", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												case "Check-in Instructions":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "checkin", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												case "Check-out Instructions":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "checkout", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												case "Athens Airport →Apartment":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "athenstoairport", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												case "Piraeus Port → Apartment":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "piraeustoairport", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												case "Stove / Child Lock":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "childlock", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												case "Taxi / Ride Apps":
													if err := sendWhatsAppTemplateMessage(token, fromPhone, "taxi", "en"); err != nil {
														fmt.Println("Error:", err)
													}
												default:
													log.Printf("Unknown button pressed: %v (text: %v)\n", payload, text)
												}
											}
										}
									}
								}
							}

							// --- Handle SENT / DELIVERED / READ events ---
							if statuses, ok := value["statuses"].([]any); ok {
								for _, s := range statuses {
									if statusMap, ok := s.(map[string]any); ok {
										status := statusMap["status"]
										recipientID := statusMap["recipient_id"]
										timestamp := statusMap["timestamp"]
										// convert recipientID to string
										recipientIDStr, _ := recipientID.(string)
										if status == "failed" || status == "delivered" {
											continue
										}
										// Check cache to avoid duplicate logs
										if _, found := cache.Get(recipientIDStr); !found {
											cache.Set(recipientIDStr, true, time.Hour*24) // cache for 24 hours
											handleGreeting(recipientIDStr)
											time.Sleep(1 * time.Second) // slight delay to avoid rate limits
										}

										log.Printf("Message status: %v | Recipient ID: %v | Timestamp: %v\n",
											status, recipientID, timestamp)
									}
								}
							}
						}
					}
				}
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("EVENT_RECEIVED"))
}

func sendWhatsAppTemplateMessage(token, to, templateName, languageCode string) error {
	url := "https://graph.facebook.com/v22.0/776012848931729/messages"

	// Build the payload
	msg := TemplateMessage{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "template",
	}
	msg.Template.Name = templateName
	msg.Template.Language.Code = languageCode

	payloadBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Message sent successfully!")
	} else {
		return fmt.Errorf("failed with status: %s", resp.Status)
	}

	return nil
}

type ImageMessage struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Image            struct {
		Link string `json:"link"`
	} `json:"image"`
}

func handleGreetingRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	handleGreeting("919891594807")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Greeting messages sent"))
}

func handleGreeting(recipientIDStr string) {
	token = os.Getenv("VERIFICATION_TOKEN")
	// keybox
	err := sendImageTemplate(recipientIDStr, token, "kebox_image", "https://i.postimg.cc/x8w7Drdy/4.jpg")
	if err != nil {
		fmt.Println("Error sending image message:", err)
	}
	// how to send code
	err = sendImageTemplate(recipientIDStr, token, "image_template", "https://i.postimg.cc/rpMBwL9F/3.png")
	if err != nil {
		fmt.Println("Error sending template message:", err)
	}
	err = sendWhatsAppTemplateMessage(token, recipientIDStr, "welcome_athens_new", "en")
	if err != nil {
		fmt.Println("Error sending template message:", err)
	}
}

func sendWhatsAppImageMessage(accessToken, recipient, imageURL string) error {
	url := "https://graph.facebook.com/v22.0/776012848931729/messages"
	// Prepare message payload
	payload := ImageMessage{
		MessagingProduct: "whatsapp",
		To:               recipient,
		Type:             "image",
	}
	payload.Image.Link = imageURL

	// Convert to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to send message, status: %s", resp.Status)
	}

	fmt.Println("✅ WhatsApp image message sent successfully!")
	return nil
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Server is healthy"))
}

func sendImageTemplate(toNumber, token, templateName, imageURL string) error {
	phoneNumberID := "776012848931729"
	msg := TemplateMessage{
		MessagingProduct: "whatsapp",
		To:               toNumber,
		Type:             "template",
	}
	msg.Template.Name = templateName
	msg.Template.Language.Code = "en"

	// Add image header component
	msg.Template.Components = []struct {
		Type       string `json:"type"`
		Parameters []struct {
			Type  string `json:"type"`
			Image struct {
				Link string `json:"link,omitempty"`
			} `json:"image,omitempty"`
		} `json:"parameters,omitempty"`
	}{
		{
			Type: "header",
			Parameters: []struct {
				Type  string `json:"type"`
				Image struct {
					Link string `json:"link,omitempty"`
				} `json:"image,omitempty"`
			}{
				{
					Type: "image",
					Image: struct {
						Link string `json:"link,omitempty"`
					}{
						Link: imageURL,
					},
				},
			},
		},
	}

	// Marshal payload
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Make request
	url := fmt.Sprintf("https://graph.facebook.com/v20.0/%s/messages", phoneNumberID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("request failed with status: %s", resp.Status)
	}

	fmt.Println("Message sent successfully!")
	return nil
}

func main() {
	cache := New()

	http.HandleFunc("/webhooks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGetWebhook(w, r)
		case http.MethodPost:
			handlePostWebhook(w, r, cache)
		case http.MethodDelete:
			handleGreetingRequest(w, r)
		case http.MethodHead:
			handleHealthCheck(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server started at :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
