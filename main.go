package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	// Configure logging
	logLevel := slog.LevelInfo
	
	// Support both DEBUG=true and LOG_LEVEL=debug/info/warn/error
	if os.Getenv("DEBUG") == "true" {
		logLevel = slog.LevelDebug
	} else if level := os.Getenv("LOG_LEVEL"); level != "" {
		switch level {
		case "debug":
			logLevel = slog.LevelDebug
		case "info":
			logLevel = slog.LevelInfo
		case "warn":
			logLevel = slog.LevelWarn
		case "error":
			logLevel = slog.LevelError
		}
	}
	
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	discordWebhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if discordWebhookURL == "" {
		slog.Error("DISCORD_WEBHOOK_URL environment variable is required")
		os.Exit(1)
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
		
		// Read raw request body
		bodyBytes, err := io.ReadAll(r.Body)
		must(err, http.StatusBadRequest, "Failed to read request body")
		
		// Debug logging
		slog.Debug("Received webhook request", "body", string(bodyBytes))
		
		// Decode payload
		var payload grafana.WebhookPayload
		must(json.Unmarshal(bodyBytes, &payload), http.StatusBadRequest, "Invalid request body")

		// Record alert metrics
		for _, alert := range payload.Alerts {
			severity := alert.Labels["severity"]
			if severity == "" {
				severity = "none"
			}
			metrics.RecordAlert(alert.Status, severity)
		}

		// Transform and send to Discord
		discordMsgs := transformer.GrafanaToDiscord(&payload)
		for _, discordMsg := range discordMsgs {
			discordStart := time.Now()
			err = webhook.Send(discordMsg)
			metrics.RecordDiscordSend(err == nil, time.Since(discordStart))
			must(err, http.StatusInternalServerError, "Failed to forward to Discord")
		}

		// Success
		metrics.AlertsProcessedTotal.Inc()
		slog.Info("Successfully forwarded alerts",
			"count", len(payload.Alerts),
			"status", payload.Status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
		metrics.RecordHTTPRequest("/webhook", "POST", strconv.Itoa(http.StatusOK), time.Since(start))
		metrics.RecordWebhookRequest("success")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}, "/webhook"))

	slog.Info("Server starting", "port", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), router); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
