package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pretty-discord-alerts/pkg/discord"
	"github.com/pretty-discord-alerts/pkg/grafana"
	"github.com/pretty-discord-alerts/pkg/transformer"
)

func main() {
	discordWebhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if discordWebhookURL == "" {
		log.Fatal("DISCORD_WEBHOOK_URL environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8888"
	}

	webhook := discord.NewWebhook(discordWebhookURL)
	router := http.NewServeMux()

	router.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		var payload grafana.WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			log.Printf("Failed to decode webhook payload: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		discordMsg := transformer.GrafanaToDiscord(&payload)

		if err := webhook.Send(discordMsg); err != nil {
			log.Printf("Failed to send to Discord: %v", err)
			http.Error(w, "Failed to forward to Discord", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully forwarded alert: %d firing, status: %s", len(payload.Alerts), payload.Status)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), router); err != nil {
		log.Fatal(err)
	}
}
