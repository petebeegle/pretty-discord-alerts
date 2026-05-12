package transformer

import (
	"strings"
	"testing"
	"time"

	"github.com/pretty-discord-alerts/pkg/discord"
	"github.com/pretty-discord-alerts/pkg/grafana"
)

func TestGrafanaToDiscord(t *testing.T) {
	started := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)
	ended := time.Date(2026, 2, 2, 12, 5, 0, 0, time.UTC)

	tests := []struct {
		name          string
		alert         grafana.Alert
		wantTitle     string
		wantColor     int
		wantTimestamp string
		wantFields    []string
	}{
		{
			name: "critical firing alert",
			alert: grafana.Alert{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighCPU",
					"namespace": "production",
					"pod":       "api-0",
					"severity":  "critical",
				},
				Annotations: map[string]string{
					"summary":     "CPU saturation is high",
					"description": "api-0 has exceeded the alert threshold.",
					"values":      "B=91.2, C=1",
				},
				GeneratorURL: "https://monitoring.example.com/d/api",
				StartsAt:     started,
			},
			wantTitle:     "Critical monitor triggered",
			wantColor:     colorFiring,
			wantTimestamp: started.Format(time.RFC3339),
			wantFields:    []string{"Summary", "Scope", "Values", "Status", "Actions"},
		},
		{
			name: "warning firing alert",
			alert: grafana.Alert{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "DiskPressure",
					"node":      "talos-1",
					"severity":  "warning",
				},
				Annotations: map[string]string{
					"summary": "Disk pressure is elevated",
				},
				StartsAt: started,
			},
			wantTitle:     "Warning monitor triggered",
			wantColor:     colorWarning,
			wantTimestamp: started.Format(time.RFC3339),
			wantFields:    []string{"Summary", "Scope", "Status", "Actions"},
		},
		{
			name: "resolved alert uses recovery title and end timestamp",
			alert: grafana.Alert{
				Status: "resolved",
				Labels: map[string]string{
					"alertname": "HighCPU",
					"severity":  "critical",
				},
				Annotations: map[string]string{
					"summary": "CPU saturation returned to normal",
				},
				StartsAt: started,
				EndsAt:   ended,
			},
			wantTitle:     "Monitor recovered",
			wantColor:     colorResolved,
			wantTimestamp: ended.Format(time.RFC3339),
			wantFields:    []string{"Summary", "Status", "Actions"},
		},
		{
			name: "notification severity stays neutral",
			alert: grafana.Alert{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "Player Login",
					"severity":  "notification",
				},
				Annotations: map[string]string{
					"summary": "Player logged in",
				},
				StartsAt: started,
			},
			wantTitle:     "Notification",
			wantColor:     colorNotification,
			wantTimestamp: started.Format(time.RFC3339),
			wantFields:    []string{"Summary", "Status", "Actions"},
		},
		{
			name: "missing optional fields remains valid",
			alert: grafana.Alert{
				Status:      "firing",
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			wantTitle:  "Warning monitor triggered",
			wantColor:  colorWarning,
			wantFields: []string{"Summary", "Status", "Actions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := GrafanaToDiscord(&grafana.WebhookPayload{
				Status:      tt.alert.Status,
				ExternalURL: "https://monitoring.example.com",
				Alerts:      []grafana.Alert{tt.alert},
			})

			if len(msgs) != 1 {
				t.Fatalf("message count = %d, want 1", len(msgs))
			}

			msg := msgs[0]
			if msg.Username != "Grafana" {
				t.Errorf("username = %q, want Grafana", msg.Username)
			}
			if len(msg.Embeds) != 1 {
				t.Fatalf("embed count = %d, want 1", len(msg.Embeds))
			}

			embed := msg.Embeds[0]
			if embed.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", embed.Title, tt.wantTitle)
			}
			if embed.Color != tt.wantColor {
				t.Errorf("color = %d, want %d", embed.Color, tt.wantColor)
			}
			if embed.Type != "rich" {
				t.Errorf("type = %q, want rich", embed.Type)
			}
			if embed.Timestamp != tt.wantTimestamp {
				t.Errorf("timestamp = %q, want %q", embed.Timestamp, tt.wantTimestamp)
			}
			if embed.Footer == nil || embed.Footer.Text != "Grafana monitor" {
				t.Fatalf("footer = %#v, want Grafana monitor", embed.Footer)
			}

			for _, fieldName := range tt.wantFields {
				if fieldByName(embed.Fields, fieldName) == nil {
					t.Errorf("missing field %q in %#v", fieldName, embed.Fields)
				}
			}
		})
	}
}

func TestGrafanaToDiscordMultipleAlerts(t *testing.T) {
	msgs := GrafanaToDiscord(&grafana.WebhookPayload{
		Status:      "firing",
		ExternalURL: "https://monitoring.example.com",
		Alerts: []grafana.Alert{
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "Alert1", "severity": "critical"},
				Annotations: map[string]string{"summary": "First alert"},
			},
			{
				Status:      "firing",
				Labels:      map[string]string{"alertname": "Alert2", "severity": "warning"},
				Annotations: map[string]string{"summary": "Second alert"},
			},
		},
	})

	if len(msgs) != 2 {
		t.Fatalf("message count = %d, want 2", len(msgs))
	}
}

func TestBuildFieldsContent(t *testing.T) {
	started := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)
	alert := grafana.Alert{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "HighCPU",
			"namespace": "production",
			"pod":       "api-0",
			"instance":  "10.0.0.1:9100",
			"severity":  "critical",
		},
		Annotations: map[string]string{
			"summary":     "CPU saturation is high",
			"description": "api-0 has exceeded the alert threshold.",
			"values":      "B=91.2, C=1",
		},
		GeneratorURL: "https://monitoring.example.com/d/api",
		StartsAt:     started,
	}

	fields := buildFields(alert, "https://monitoring.example.com")

	summary := fieldByName(fields, "Summary")
	if summary == nil || !strings.Contains(summary.Value, "CPU saturation is high") || !strings.Contains(summary.Value, "api-0 has exceeded") {
		t.Fatalf("summary field = %#v", summary)
	}

	scope := fieldByName(fields, "Scope")
	if scope == nil || !strings.Contains(scope.Value, "`namespace:production`") || !strings.Contains(scope.Value, "`pod:api-0`") || !strings.Contains(scope.Value, "`instance:10.0.0.1:9100`") {
		t.Fatalf("scope field = %#v", scope)
	}

	values := fieldByName(fields, "Values")
	if values == nil || values.Value != "B=91.2, C=1" {
		t.Fatalf("values field = %#v", values)
	}

	status := fieldByName(fields, "Status")
	if status == nil || !strings.Contains(status.Value, "Firing") || !strings.Contains(status.Value, started.Format(time.RFC3339)) {
		t.Fatalf("status field = %#v", status)
	}

	actions := fieldByName(fields, "Actions")
	if actions == nil || !strings.Contains(actions.Value, "[Source]") || !strings.Contains(actions.Value, "[Silence]") {
		t.Fatalf("actions field = %#v", actions)
	}
}

func TestBuildFieldsOmitMissingOptionalFields(t *testing.T) {
	fields := buildFields(grafana.Alert{
		Status:      "firing",
		Labels:      map[string]string{"severity": "warning"},
		Annotations: map[string]string{},
	}, "")

	if fieldByName(fields, "Summary") == nil {
		t.Fatal("missing summary field")
	}
	if fieldByName(fields, "Scope") != nil {
		t.Fatal("scope field should be omitted")
	}
	if fieldByName(fields, "Values") != nil {
		t.Fatal("values field should be omitted")
	}
	if fieldByName(fields, "Actions") != nil {
		t.Fatal("actions field should be omitted")
	}
}

func TestBuildSilenceURL(t *testing.T) {
	got := buildSilenceURL("https://monitoring.example.com", map[string]string{
		"alertname": "High CPU",
		"namespace": "production",
	})

	for _, want := range []string{
		"https://monitoring.example.com/alerting/silence/new?",
		"alertmanager=grafana",
		"matcher=alertname%3DHigh+CPU",
		"matcher=namespace%3Dproduction",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildSilenceURL() = %q, missing %q", got, want)
		}
	}
}

func TestDiscordFieldValuesStayWithinLimit(t *testing.T) {
	longSummary := strings.Repeat("a", maxFieldValueLength+50)

	fields := buildFields(grafana.Alert{
		Status: "firing",
		Labels: map[string]string{
			"severity": "critical",
		},
		Annotations: map[string]string{
			"summary": longSummary,
		},
	}, "")

	for _, field := range fields {
		if len(field.Value) > maxFieldValueLength {
			t.Fatalf("field %q length = %d, want <= %d", field.Name, len(field.Value), maxFieldValueLength)
		}
	}
}

func fieldByName(fields []discord.EmbedField, name string) *discord.EmbedField {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}

	return nil
}
