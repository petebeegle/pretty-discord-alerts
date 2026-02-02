package transformer

import (
	"fmt"
	"time"

	"github.com/pretty-discord-alerts/pkg/discord"
	"github.com/pretty-discord-alerts/pkg/grafana"
)

const (
	colorFiring   = 15158332 // Red
	colorWarning  = 16776960 // Yellow
	colorResolved = 3066993  // Green
	maxAlerts     = 10
)

// GrafanaToDiscord transforms a Grafana webhook payload to a Discord message
func GrafanaToDiscord(payload *grafana.WebhookPayload) discord.Message {
	var firingCount, resolvedCount int
	var severity string

	// Count alerts and determine severity
	for _, alert := range payload.Alerts {
		switch alert.Status {
		case "firing":
			firingCount++
			if sev := alert.Labels["severity"]; sev == "critical" {
				severity = "critical"
			} else if sev == "warning" && severity != "critical" {
				severity = "warning"
			}
		case "resolved":
			resolvedCount++
		}
	}

	embed := discord.Embed{
		Color:       getColor(firingCount, resolvedCount, severity),
		Title:       getTitle(firingCount, resolvedCount, severity),
		Description: getDescription(firingCount, resolvedCount),
		Fields:      buildFields(payload.Alerts),
		Footer: &discord.EmbedFooter{
			Text:    "Grafana Alerts",
			IconURL: "https://grafana.com/static/assets/img/fav32.png",
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return discord.Message{Embeds: []discord.Embed{embed}}
}

func getTitle(firingCount, resolvedCount int, severity string) string {
	if firingCount > 0 {
		if severity == "critical" {
			return "🔥 Critical Alerts Firing"
		}
		return "⚠️ Warning Alerts Firing"
	}
	if resolvedCount > 0 {
		return "✅ Alerts Resolved"
	}
	return "📊 Alert Status Update"
}

func getDescription(firingCount, resolvedCount int) string {
	if firingCount > 0 && resolvedCount > 0 {
		return fmt.Sprintf("%d alert(s) firing, %d alert(s) resolved", firingCount, resolvedCount)
	}
	if firingCount > 0 {
		return fmt.Sprintf("%d alert(s) firing", firingCount)
	}
	if resolvedCount > 0 {
		return fmt.Sprintf("%d alert(s) resolved", resolvedCount)
	}
	return "No alerts"
}

func getColor(firingCount, resolvedCount int, severity string) int {
	if firingCount > 0 {
		if severity == "critical" {
			return colorFiring
		}
		return colorWarning
	}
	if resolvedCount > 0 {
		return colorResolved
	}
	return colorWarning
}

func buildFields(alerts []grafana.Alert) []discord.EmbedField {
	fields := make([]discord.EmbedField, 0, len(alerts))

	for i, alert := range alerts {
		if i >= maxAlerts {
			fields = append(fields, discord.EmbedField{
				Name:   "Additional Alerts",
				Value:  fmt.Sprintf("... and %d more alert(s)", len(alerts)-maxAlerts),
				Inline: false,
			})
			break
		}

		name := alert.Labels["alertname"]
		if name == "" {
			name = "Unknown Alert"
		}

		value := buildFieldValue(alert)
		fields = append(fields, discord.EmbedField{
			Name:   fmt.Sprintf("%d. %s", i+1, name),
			Value:  value,
			Inline: false,
		})
	}

	return fields
}

func buildFieldValue(alert grafana.Alert) string {
	var value string

	if summary := alert.Annotations["summary"]; summary != "" {
		value += fmt.Sprintf("**Summary:** %s\n", summary)
	}
	if description := alert.Annotations["description"]; description != "" {
		value += fmt.Sprintf("**Description:** %s\n", description)
	}
	if namespace := alert.Labels["namespace"]; namespace != "" {
		value += fmt.Sprintf("**Namespace:** %s\n", namespace)
	}

	emoji := "🔴"
	status := "Firing"
	if alert.Status == "resolved" {
		emoji = "✅"
		status = "Resolved"
	}

	value += fmt.Sprintf("**Status:** %s %s", emoji, status)
	return value
}
