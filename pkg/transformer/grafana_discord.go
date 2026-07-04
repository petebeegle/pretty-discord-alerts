package transformer

import (
	"fmt"
	"net/url"
	"regexp"
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
var grafanaValueStringRefPattern = regexp.MustCompile(`var='([^']+)'\s+labels=\{[^}]*\}\s+value=([^,\]\s]+)`)

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
			Description: buildDescription(alert),
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
	if severity == "" {
		severity = "warning"
	}

	status := "Firing"
	if alert.Status == "resolved" {
		status = "Recovered"
	} else if alert.Status != "firing" && alert.Status != "" {
		status = titleCase(alert.Status)
	}

	return fmt.Sprintf("%s %s: %s", status, strings.ToLower(severity), getAlertName(alert))
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
			Name:   "Impact",
			Value:  truncateFieldValue(buildImpact(alert)),
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
			Name:   "Value",
			Value:  truncateFieldValue(values),
			Inline: true,
		})
	}

	fields = append(fields, discord.EmbedField{
		Name:   "Timeline",
		Value:  buildTimeline(alert),
		Inline: true,
	})

	fields = append(fields, discord.EmbedField{
		Name:   "Next step",
		Value:  truncateFieldValue(buildNextStep(alert)),
		Inline: false,
	})

	if links := buildLinks(alert, externalURL); links != "" {
		fields = append(fields, discord.EmbedField{
			Name:   "Links",
			Value:  truncateFieldValue(links),
			Inline: false,
		})
	}

	return fields
}

func buildDescription(alert grafana.Alert) string {
	if summary := strings.TrimSpace(alert.Annotations["summary"]); summary != "" {
		return truncateFieldValue(summary)
	}

	return getAlertName(alert)
}

func buildImpact(alert grafana.Alert) string {
	if description := strings.TrimSpace(alert.Annotations["description"]); description != "" {
		return description
	}
	if summary := strings.TrimSpace(alert.Annotations["summary"]); summary != "" {
		return summary
	}

	return "No impact description provided."
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

	if value := parseObservedValueAnnotation(alert.Annotations["values"]); value != "" {
		return value
	}

	return parseObservedValueAnnotation(alert.ValueString)
}

func parseObservedValueAnnotation(value string) string {
	for _, match := range grafanaValueStringRefPattern.FindAllStringSubmatch(value, -1) {
		if len(match) == 3 && match[1] == "B" {
			return strings.TrimSpace(match[2])
		}
	}

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

func buildTimeline(alert grafana.Alert) string {
	status := titleCase(alert.Status)
	if status == "" {
		status = "Unknown"
	}

	if ts := statusTime(alert); !ts.IsZero() {
		var lines []string
		if alert.Status == "resolved" {
			lines = append(lines, status, fmt.Sprintf("Ended: `%s`", ts.UTC().Format(time.RFC3339)))
			if !alert.StartsAt.IsZero() && alert.EndsAt.After(alert.StartsAt) {
				lines = append(lines, fmt.Sprintf("Duration: `%s`", alert.EndsAt.Sub(alert.StartsAt).Round(time.Second)))
			}
			return strings.Join(lines, "\n")
		}

		return fmt.Sprintf("%s\nStarted: `%s`", status, ts.UTC().Format(time.RFC3339))
	}

	return status
}

func buildNextStep(alert grafana.Alert) string {
	if runbook := strings.TrimSpace(alert.Annotations["runbook"]); runbook != "" {
		return runbook
	}

	return "Open the source link and inspect the affected scope."
}

func buildLinks(alert grafana.Alert, externalURL string) string {
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
