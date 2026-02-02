package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pretty-discord-alerts/pkg/discord"
	"github.com/pretty-discord-alerts/pkg/grafana"
	"github.com/pretty-discord-alerts/pkg/metrics"
	"github.com/pretty-discord-alerts/pkg/middleware"
	"github.com/pretty-discord-alerts/pkg/transformer"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// must panics with an HTTPError if err is not nil
func must(err error, status int, message string) {
	if err != nil {
		panic(&middleware.HTTPError{Status: status, Message: message, Cause: err})
	}
}

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

	// Prometheus metrics endpoint
	router.Handle("GET /metrics", promhttp.Handler())

	router.HandleFunc("POST /webhook", middleware.RecoverMiddleware(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Decode payload
		var payload grafana.WebhookPayload
		must(json.NewDecoder(r.Body).Decode(&payload), http.StatusBadRequest, "Invalid request body")

		// Record alert metrics
		for _, alert := range payload.Alerts {
			severity := alert.Labels["severity"]
			if severity == "" {
				severity = "none"
			}
			metrics.RecordAlert(alert.Status, severity)
		}

		// Transform and send to Discord
		discordMsg := transformer.GrafanaToDiscord(&payload)
		discordStart := time.Now()
		err := webhook.Send(discordMsg)
		metrics.RecordDiscordSend(err == nil, time.Since(discordStart))
		must(err, http.StatusInternalServerError, "Failed to forward to Discord")

		// Success
		metrics.AlertsProcessedTotal.Inc()
		log.Printf("Successfully forwarded alert: %d firing, status: %s", len(payload.Alerts), payload.Status)
		metrics.RecordHTTPRequest("/webhook", "POST", strconv.Itoa(http.StatusOK), time.Since(start))
		metrics.RecordWebhookRequest("success")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}, "/webhook"))

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), router); err != nil {
		log.Fatal(err)
	}
}
