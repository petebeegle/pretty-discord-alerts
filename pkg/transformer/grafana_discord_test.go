package transformer

import (
	"testing"
	"time"

	"github.com/pretty-discord-alerts/pkg/grafana"
)

func TestGrafanaToDiscord(t *testing.T) {
	tests := []struct {
		name       string
		payload    *grafana.WebhookPayload
		wantTitle  string
		wantColor  int
		wantCount  int
	}{
		{
			name: "single firing alert",
			payload: &grafana.WebhookPayload{
				Status:      "firing",
				ExternalURL: "https://monitoring.example.com",
				Alerts: []grafana.Alert{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname":      "TestAlert",
							"grafana_folder": "Test Folder",
							"instance":       "Grafana",
							"severity":       "critical",
						},
						Annotations: map[string]string{
							"summary": "Notification test",
							"values":  "B=22, C=1",
						},
						GeneratorURL: "https://monitoring.example.com/d/dashboard_uid",
						StartsAt:     time.Now(),
					},
				},
			},
			wantTitle: "🔥 Critical Alert Firing",
			wantColor: colorFiring,
			wantCount: 1,
		},
		{
			name: "multiple alerts",
			payload: &grafana.WebhookPayload{
				Status:      "firing",
				ExternalURL: "https://monitoring.example.com",
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
						Status: "firing",
						Labels: map[string]string{
							"alertname": "Alert2",
							"severity":  "warning",
						},
						Annotations: map[string]string{
							"summary": "Second alert",
						},
					},
				},
			},
			wantCount: 2,
		},
		{
			name: "resolved alert",
			payload: &grafana.WebhookPayload{
				Status:      "resolved",
				ExternalURL: "https://monitoring.example.com",
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
			wantTitle: "✅ Alert Resolved",
			wantColor: colorResolved,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := GrafanaToDiscord(tt.payload)

			if len(msgs) != tt.wantCount {
				t.Errorf("expected %d messages, got %d", tt.wantCount, len(msgs))
				return
			}

			if tt.wantCount == 0 {
				return
			}

			msg := msgs[0]

			if msg.Username != "Grafana" {
				t.Errorf("username = %q, want %q", msg.Username, "Grafana")
			}

			if len(msg.Embeds) != 1 {
				t.Errorf("expected 1 embed, got %d", len(msg.Embeds))
				return
			}

			embed := msg.Embeds[0]

			if tt.wantTitle != "" && embed.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", embed.Title, tt.wantTitle)
			}

			if tt.wantColor != 0 && embed.Color != tt.wantColor {
				t.Errorf("color = %d, want %d", embed.Color, tt.wantColor)
			}

			if embed.Type != "rich" {
				t.Errorf("type = %q, want %q", embed.Type, "rich")
			}

			if len(embed.Fields) != 1 {
				t.Errorf("field count = %d, want 1", len(embed.Fields))
			}

			if embed.Footer == nil {
				t.Error("footer is nil")
			} else if embed.Footer.Text != "Grafana v12.3.2" {
				t.Errorf("footer text = %q, want %q", embed.Footer.Text, "Grafana v12.3.2")
			}
		})
	}
}

func TestGetAlertTitle(t *testing.T) {
	tests := []struct {
		name  string
		alert grafana.Alert
		want  string
	}{
		{
			name: "critical firing",
			alert: grafana.Alert{
				Status: "firing",
				Labels: map[string]string{"severity": "critical"},
			},
			want: "🔥 Critical Alert Firing",
		},
		{
			name: "warning firing",
			alert: grafana.Alert{
				Status: "firing",
				Labels: map[string]string{"severity": "warning"},
			},
			want: "⚠️ Warning Alert Firing",
		},
		{
			name: "resolved",
			alert: grafana.Alert{
				Status: "resolved",
			},
			want: "✅ Alert Resolved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAlertTitle(tt.alert)
			if got != tt.want {
				t.Errorf("getAlertTitle() = %q, want %q", got, tt.want)
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
		GeneratorURL: "https://monitoring.example.com/d/dashboard",
	}

	value := buildFieldValue(alert, "https://monitoring.example.com")

	if value == "" {
		t.Error("buildFieldValue() returned empty string")
	}

	// Check that all fields are included
	expected := []string{"Test summary", "Test description", "production", "🔴", "Firing", "View Source", "Silence"}
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
