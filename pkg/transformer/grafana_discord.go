package transformer

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pretty-discord-alerts/pkg/discord"
	"github.com/pretty-discord-alerts/pkg/grafana"
)

const (
	colorFiring       = 15026253 // Red
	colorWarning      = 16756768 // Orange/yellow
	colorResolved     = 3069565  // Green
	colorNotification = 6583435  // Neutral gray/blue

	maxFieldValueLength = 1024
)

var scopeLabelKeys = []string{"namespace", "pod", "instance", "job", "service", "node", "component"}

// GrafanaToDiscord transforms a Grafana webhook payload to Discord messages (one per alert)
func GrafanaToDiscord(payload *grafana.WebhookPayload) []discord.Message {
	messages := make([]discord.Message, 0, len(payload.Alerts))

	for _, alert := range payload.Alerts {
		alertingURL := payload.ExternalURL
		if alertingURL != "" {
			alertingURL = strings.TrimSuffix(alertingURL, "/") + "/alerting/list"
		}

		embed := discord.Embed{
			Title:       getAlertTitle(alert),
			Description: getAlertName(alert),
			Type:        "rich",
			URL:         alertingURL,
			Color:       getAlertColor(alert),
			Fields:      buildFields(alert, payload.ExternalURL),
			Footer: &discord.EmbedFooter{
				Text:    "Grafana monitor",
				IconURL: "https://grafana.com/static/assets/img/fav32.png",
			},
			Timestamp: getAlertTimestamp(alert),
		}

		messages = append(messages, discord.Message{
			Username: "Grafana",
			Embeds:   []discord.Embed{embed},
		})
	}

	return messages
}

func getAlertColor(alert grafana.Alert) int {
	severity := alert.Labels["severity"]

	if severity == "notification" || severity == "info" {
		return colorNotification
	}

	if alert.Status == "resolved" {
		return colorResolved
	}

	if severity == "critical" {
		return colorFiring
	}

	return colorWarning
}

func getAlertTitle(alert grafana.Alert) string {
	severity := alert.Labels["severity"]

	if severity == "notification" || severity == "info" {
		return "Notification"
	}

	if alert.Status == "firing" {
		if severity == "critical" {
			return "Critical monitor triggered"
		}
		return "Warning monitor triggered"
	}

	return "Monitor recovered"
}

func getAlertName(alert grafana.Alert) string {
	if alertName := alert.Labels["alertname"]; alertName != "" {
		return alertName
	}

	return "Grafana alert"
}

func buildFields(alert grafana.Alert, externalURL string) []discord.EmbedField {
	fields := []discord.EmbedField{
		{
			Name:   "Summary",
			Value:  truncateFieldValue(buildSummary(alert)),
			Inline: false,
		},
	}

	if scope := buildScope(alert); scope != "" {
		fields = append(fields, discord.EmbedField{
			Name:   "Scope",
			Value:  truncateFieldValue(scope),
			Inline: false,
		})
	}

	if values := buildObservedValue(alert); values != "" {
		fields = append(fields, discord.EmbedField{
			Name:   "Observed value",
			Value:  truncateFieldValue(values),
			Inline: false,
		})
	}

	fields = append(fields, discord.EmbedField{
		Name:   "Status",
		Value:  buildStatus(alert),
		Inline: true,
	})

	if actions := buildActions(alert, externalURL); actions != "" {
		fields = append(fields, discord.EmbedField{
			Name:   "Actions",
			Value:  truncateFieldValue(actions),
			Inline: false,
		})
	}

	return fields
}

func buildSummary(alert grafana.Alert) string {
	var lines []string

	if summary := strings.TrimSpace(alert.Annotations["summary"]); summary != "" {
		lines = append(lines, summary)
	}
	if description := strings.TrimSpace(alert.Annotations["description"]); description != "" {
		lines = append(lines, description)
	}

	if len(lines) == 0 {
		return "No summary provided."
	}

	return strings.Join(lines, "\n")
}

func buildScope(alert grafana.Alert) string {
	var labels []string

	for _, key := range scopeLabelKeys {
		if value := strings.TrimSpace(alert.Labels[key]); value != "" {
			labels = append(labels, fmt.Sprintf("`%s:%s`", key, value))
		}
	}

	return strings.Join(labels, " ")
}

func buildObservedValue(alert grafana.Alert) string {
	if value, ok := alert.Values["B"]; ok {
		return formatFloat(value)
	}

	return parseObservedValueAnnotation(alert.Annotations["values"])
}

func parseObservedValueAnnotation(value string) string {
	for _, part := range strings.Split(value, ",") {
		key, val, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok || strings.TrimSpace(key) != "B" {
			continue
		}

		return strings.TrimSpace(val)
	}

	return ""
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func buildStatus(alert grafana.Alert) string {
	status := titleCase(alert.Status)
	if status == "" {
		status = "Unknown"
	}

	if ts := statusTime(alert); !ts.IsZero() {
		if alert.Status == "resolved" {
			return fmt.Sprintf("%s\nEnded: `%s`", status, ts.UTC().Format(time.RFC3339))
		}

		return fmt.Sprintf("%s\nStarted: `%s`", status, ts.UTC().Format(time.RFC3339))
	}

	return status
}

func buildActions(alert grafana.Alert, externalURL string) string {
	var links []string

	if alert.GeneratorURL != "" {
		links = append(links, fmt.Sprintf("[Source](%s)", alert.GeneratorURL))
	}
	if externalURL != "" {
		links = append(links, fmt.Sprintf("[Silence](%s)", buildSilenceURL(externalURL, alert.Labels)))
	}

	return strings.Join(links, " • ")
}

func getAlertTimestamp(alert grafana.Alert) string {
	ts := statusTime(alert)
	if ts.IsZero() {
		return ""
	}

	return ts.UTC().Format(time.RFC3339)
}

func statusTime(alert grafana.Alert) time.Time {
	if alert.Status == "resolved" && !alert.EndsAt.IsZero() {
		return alert.EndsAt
	}

	if !alert.StartsAt.IsZero() {
		return alert.StartsAt
	}

	return time.Time{}
}

func buildSilenceURL(externalURL string, labels map[string]string) string {
	baseURL := strings.TrimSuffix(externalURL, "/")
	silenceURL, err := url.Parse(baseURL + "/alerting/silence/new")
	if err != nil {
		return baseURL + "/alerting/silence/new"
	}

	query := silenceURL.Query()
	query.Set("alertmanager", "grafana")

	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		query.Add("matcher", fmt.Sprintf("%s=%s", key, labels[key]))
	}

	if strings.Contains(externalURL, "orgId=") {
		query.Set("orgId", "1")
	}

	silenceURL.RawQuery = query.Encode()
	return silenceURL.String()
}

func truncateFieldValue(value string) string {
	if len(value) <= maxFieldValueLength {
		return value
	}

	return value[:maxFieldValueLength-3] + "..."
}

func titleCase(value string) string {
	if value == "" {
		return ""
	}

	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}
