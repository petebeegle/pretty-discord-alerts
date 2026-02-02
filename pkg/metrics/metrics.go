package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	// Webhook-specific metrics
	WebhookRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_requests_total",
			Help: "Total number of webhook requests received",
		},
		[]string{"status"},
	)

	WebhookDiscordSendTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_discord_send_total",
			Help: "Total number of Discord webhook sends",
		},
		[]string{"status"},
	)

	WebhookDiscordSendDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "webhook_discord_send_duration_seconds",
			Help:    "Duration of Discord webhook sends in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	// Alert metrics
	AlertsReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alerts_received_total",
			Help: "Total number of alerts received from Grafana",
		},
		[]string{"status", "severity"},
	)

	AlertsProcessedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alerts_processed_total",
			Help: "Total number of alerts successfully processed",
		},
	)
)

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(path, method, status string, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(path, method, status).Inc()
	HTTPRequestDuration.WithLabelValues(path, method).Observe(duration.Seconds())
}

// RecordWebhookRequest increments the webhook request counter
func RecordWebhookRequest(status string) {
	WebhookRequestsTotal.WithLabelValues(status).Inc()
}

// RecordDiscordSend records metrics for Discord webhook sends
func RecordDiscordSend(success bool, duration time.Duration) {
	status := "success"
	if !success {
		status = "failure"
	}
	WebhookDiscordSendTotal.WithLabelValues(status).Inc()
	WebhookDiscordSendDuration.Observe(duration.Seconds())
}

// RecordAlert increments the alert counter
func RecordAlert(alertStatus, severity string) {
	AlertsReceivedTotal.WithLabelValues(alertStatus, severity).Inc()
}
