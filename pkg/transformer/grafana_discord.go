package transformer

import (
	"fmt"
	"strings"

	"github.com/pretty-discord-alerts/pkg/discord"
	"github.com/pretty-discord-alerts/pkg/grafana"
)

const (
	colorFiring   = 14037554 // Red (Grafana default)
	colorWarning  = 16776960 // Yellow
	colorResolved = 3066993  // Green
)

// GrafanaToDiscord transforms a Grafana webhook payload to Discord messages (one per alert)
func GrafanaToDiscord(payload *grafana.WebhookPayload) []discord.Message {
	messages := make([]discord.Message, 0, len(payload.Alerts))

	for _, alert := range payload.Alerts {
		// Determine severity and color
		severity := alert.Labels["severity"]
		color := colorResolved
		if alert.Status == "firing" {
			if severity == "critical" {
				color = colorFiring
			} else {
				color = colorWarning
			}
		}

		// Get alerting URL from external URL
		alertingURL := payload.ExternalURL
		if alertingURL != "" {
			alertingURL = strings.TrimSuffix(alertingURL, "/") + "/alerting/list"
		}

		// Build title
		title := getAlertTitle(alert)

		// Build field value
		fieldValue := buildFieldValue(alert, payload.ExternalURL)

		embed := discord.Embed{
			Title:       title,
			Description: "",
			Type:        "rich",
			URL:         alertingURL,
			Color:       color,
			Fields: []discord.EmbedField{
				{
					Name:   alert.Labels["alertname"],
					Value:  fieldValue,
					Inline: false,
				},
			},
			Footer: &discord.EmbedFooter{
				Text:    "Grafana v12.3.2",
				IconURL: "https://grafana.com/static/assets/img/fav32.png",
			},
		}

		messages = append(messages, discord.Message{
			Username: "Grafana",
			Embeds:   []discord.Embed{embed},
		})
	}

	return messages
}

func getAlertTitle(alert grafana.Alert) string {
	severity := alert.Labels["severity"]
	
	if alert.Status == "firing" {
		if severity == "critical" {
			return "🔥 Critical Alert Firing"
		}
		return "⚠️ Warning Alert Firing"
	}
	return "✅ Alert Resolved"
}

func buildFieldValue(alert grafana.Alert, externalURL string) string {
	var value string

	if summary := alert.Annotations["summary"]; summary != "" {
		value += fmt.Sprintf("**Summary:** %s\n", summary)
	}
	if description := alert.Annotations["description"]; description != "" {
		value += fmt.Sprintf("**Description:** %s\n", description)
	}
	if values := alert.Annotations["values"]; values != "" {
		value += fmt.Sprintf("**Query Results:** %s\n", values)
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

	value += fmt.Sprintf("**Status:** %s %s\n", emoji, status)

	// Add action links
	var links []string
	if alert.GeneratorURL != "" {
		links = append(links, fmt.Sprintf("[View Source](%s)", alert.GeneratorURL))
	}
	if externalURL != "" {
		silenceURL := buildSilenceURL(externalURL, alert.Labels)
		links = append(links, fmt.Sprintf("[Silence](%s)", silenceURL))
	}
	
	if len(links) > 0 {
		value += "\n" + strings.Join(links, " • ")
	}

	return value
}

func buildSilenceURL(externalURL string, labels map[string]string) string {
	baseURL := strings.TrimSuffix(externalURL, "/")
	silenceURL := baseURL + "/alerting/silence/new?alertmanager=grafana"
	
	for key, value := range labels {
		// URL encode the matcher
		silenceURL += fmt.Sprintf("&matcher=%s%%3D%s", key, strings.ReplaceAll(value, " ", "+"))
	}
	
	// Add orgId if present in external URL
	if strings.Contains(externalURL, "orgId=") {
		silenceURL += "&orgId=1"
	}
	
	return silenceURL
}
