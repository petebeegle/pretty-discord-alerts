package transformer

import (
	"testing"
	"time"

	"github.com/pretty-discord-alerts/pkg/grafana"
)

func TestGrafanaToDiscord(t *testing.T) {
	tests := []struct {
		name          string
		payload       *grafana.WebhookPayload
		wantTitle     string
		wantColor     int
		wantFieldCount int
	}{
		{
			name: "single firing critical alert",
			payload: &grafana.WebhookPayload{
				Status: "firing",
				Alerts: []grafana.Alert{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "HighCPU",
							"severity":  "critical",
							"namespace": "production",
						},
						Annotations: map[string]string{
							"summary":     "CPU is high",
							"description": "CPU usage above 90%",
						},
						StartsAt: time.Now(),
					},
				},
			},
			wantTitle:      "🔥 Critical Alerts Firing",
			wantColor:      colorFiring,
			wantFieldCount: 1,
		},
		{
			name: "single firing warning alert",
			payload: &grafana.WebhookPayload{
				Status: "firing",
				Alerts: []grafana.Alert{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "HighMemory",
							"severity":  "warning",
						},
						Annotations: map[string]string{
							"summary": "Memory is high",
						},
						StartsAt: time.Now(),
					},
				},
			},
			wantTitle:      "⚠️ Warning Alerts Firing",
			wantColor:      colorWarning,
			wantFieldCount: 1,
		},
		{
			name: "resolved alert",
			payload: &grafana.WebhookPayload{
				Status: "resolved",
				Alerts: []grafana.Alert{
					{
						Status: "resolved",
						Labels: map[string]string{
							"alertname": "HighCPU",
						},
						Annotations: map[string]string{
							"summary": "CPU is normal",
						},
						EndsAt: time.Now(),
					},
				},
			},
			wantTitle:      "✅ Alerts Resolved",
			wantColor:      colorResolved,
			wantFieldCount: 1,
		},
		{
			name: "multiple alerts",
			payload: &grafana.WebhookPayload{
				Status: "firing",
				Alerts: []grafana.Alert{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "Alert1",
							"severity":  "critical",
						},
						Annotations: map[string]string{
							"summary": "First alert",
						},
					},
					{
						Status: "resolved",
						Labels: map[string]string{
							"alertname": "Alert2",
						},
						Annotations: map[string]string{
							"summary": "Second alert",
						},
					},
				},
			},
			wantTitle:      "🔥 Critical Alerts Firing",
			wantColor:      colorFiring,
			wantFieldCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := GrafanaToDiscord(tt.payload)

			if len(msg.Embeds) != 1 {
				t.Errorf("expected 1 embed, got %d", len(msg.Embeds))
				return
			}

			embed := msg.Embeds[0]

			if embed.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", embed.Title, tt.wantTitle)
			}

			if embed.Color != tt.wantColor {
				t.Errorf("color = %d, want %d", embed.Color, tt.wantColor)
			}

			if len(embed.Fields) != tt.wantFieldCount {
				t.Errorf("field count = %d, want %d", len(embed.Fields), tt.wantFieldCount)
			}

			if embed.Footer == nil {
				t.Error("footer is nil")
			} else if embed.Footer.Text != "Grafana Alerts" {
				t.Errorf("footer text = %q, want %q", embed.Footer.Text, "Grafana Alerts")
			}

			if embed.Timestamp == "" {
				t.Error("timestamp is empty")
			}
		})
	}
}

func TestGetTitle(t *testing.T) {
	tests := []struct {
		name          string
		firingCount   int
		resolvedCount int
		severity      string
		want          string
	}{
		{"critical firing", 1, 0, "critical", "🔥 Critical Alerts Firing"},
		{"warning firing", 1, 0, "warning", "⚠️ Warning Alerts Firing"},
		{"resolved", 0, 1, "", "✅ Alerts Resolved"},
		{"no alerts", 0, 0, "", "📊 Alert Status Update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getTitle(tt.firingCount, tt.resolvedCount, tt.severity)
			if got != tt.want {
				t.Errorf("getTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetDescription(t *testing.T) {
	tests := []struct {
		name          string
		firingCount   int
		resolvedCount int
		want          string
	}{
		{"both", 2, 1, "2 alert(s) firing, 1 alert(s) resolved"},
		{"firing only", 3, 0, "3 alert(s) firing"},
		{"resolved only", 0, 2, "2 alert(s) resolved"},
		{"none", 0, 0, "No alerts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDescription(tt.firingCount, tt.resolvedCount)
			if got != tt.want {
				t.Errorf("getDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetColor(t *testing.T) {
	tests := []struct {
		name          string
		firingCount   int
		resolvedCount int
		severity      string
		want          int
	}{
		{"critical", 1, 0, "critical", colorFiring},
		{"warning", 1, 0, "warning", colorWarning},
		{"resolved", 0, 1, "", colorResolved},
		{"default", 0, 0, "", colorWarning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getColor(tt.firingCount, tt.resolvedCount, tt.severity)
			if got != tt.want {
				t.Errorf("getColor() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBuildFieldValue(t *testing.T) {
	alert := grafana.Alert{
		Status: "firing",
		Labels: map[string]string{
			"namespace": "production",
		},
		Annotations: map[string]string{
			"summary":     "Test summary",
			"description": "Test description",
		},
	}

	value := buildFieldValue(alert)

	if value == "" {
		t.Error("buildFieldValue() returned empty string")
	}

	// Check that all fields are included
	expected := []string{"Test summary", "Test description", "production", "🔴", "Firing"}
	for _, exp := range expected {
		if !contains(value, exp) {
			t.Errorf("buildFieldValue() missing %q in output: %q", exp, value)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsInString(s, substr))
}

func containsInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
