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
				},
				Values:       map[string]float64{"B": 91.2, "C": 1},
				ValueString:  "[ var='B' labels={} value=91.2 ], [ var='C' labels={} value=1 ]",
				GeneratorURL: "https://monitoring.example.com/d/api",
				StartsAt:     started,
			},
			wantTitle:     "Firing critical: HighCPU",
			wantColor:     colorFiring,
			wantTimestamp: started.Format(time.RFC3339),
			wantFields:    []string{"Impact", "Scope", "Value", "Timeline", "Next step", "Links"},
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
			wantTitle:     "Firing warning: DiskPressure",
			wantColor:     colorWarning,
			wantTimestamp: started.Format(time.RFC3339),
			wantFields:    []string{"Impact", "Scope", "Timeline", "Next step", "Links"},
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
			wantTitle:     "Recovered critical: HighCPU",
			wantColor:     colorResolved,
			wantTimestamp: ended.Format(time.RFC3339),
			wantFields:    []string{"Impact", "Timeline", "Next step", "Links"},
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
			wantTitle:     "Firing notification: Player Login",
			wantColor:     colorNotification,
			wantTimestamp: started.Format(time.RFC3339),
			wantFields:    []string{"Impact", "Timeline", "Next step", "Links"},
		},
		{
			name: "missing optional fields remains valid",
			alert: grafana.Alert{
				Status:      "firing",
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			wantTitle:  "Firing warning: Grafana alert",
			wantColor:  colorWarning,
			wantFields: []string{"Impact", "Timeline", "Next step"},
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
			if embed.Description == "" {
				t.Error("description should not be empty")
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
		},
		Values:       map[string]float64{"B": 91.2, "C": 1},
		ValueString:  "[ var='B' labels={} value=91.2 ], [ var='C' labels={} value=1 ]",
		GeneratorURL: "https://monitoring.example.com/d/api",
		StartsAt:     started,
	}

	fields := buildFields(alert, "https://monitoring.example.com")

	impact := fieldByName(fields, "Impact")
	if impact == nil || !strings.Contains(impact.Value, "api-0 has exceeded") || strings.Contains(impact.Value, "CPU saturation is high") {
		t.Fatalf("impact field = %#v", impact)
	}

	scope := fieldByName(fields, "Scope")
	if scope == nil || !strings.Contains(scope.Value, "`namespace:production`") || !strings.Contains(scope.Value, "`pod:api-0`") || !strings.Contains(scope.Value, "`instance:10.0.0.1:9100`") {
		t.Fatalf("scope field = %#v", scope)
	}

	observed := fieldByName(fields, "Value")
	if observed == nil || observed.Value != "91.2" {
		t.Fatalf("value field = %#v", observed)
	}
	if strings.Contains(observed.Value, "C=") || strings.Contains(observed.Value, "C") {
		t.Fatalf("observed value should not include threshold condition ref C: %#v", observed)
	}

	timeline := fieldByName(fields, "Timeline")
	if timeline == nil || !strings.Contains(timeline.Value, "Firing") || !strings.Contains(timeline.Value, started.Format(time.RFC3339)) {
		t.Fatalf("timeline field = %#v", timeline)
	}

	nextStep := fieldByName(fields, "Next step")
	if nextStep == nil || nextStep.Value != "Open the source link and inspect the affected scope." {
		t.Fatalf("next step field = %#v", nextStep)
	}

	links := fieldByName(fields, "Links")
	if links == nil || !strings.Contains(links.Value, "[Source]") || !strings.Contains(links.Value, "[Silence]") {
		t.Fatalf("links field = %#v", links)
	}
}

func TestBuildFieldsOmitMissingOptionalFields(t *testing.T) {
	fields := buildFields(grafana.Alert{
		Status:      "firing",
		Labels:      map[string]string{"severity": "warning"},
		Annotations: map[string]string{},
	}, "")

	if fieldByName(fields, "Impact") == nil {
		t.Fatal("missing impact field")
	}
	if fieldByName(fields, "Scope") != nil {
		t.Fatal("scope field should be omitted")
	}
	if fieldByName(fields, "Value") != nil {
		t.Fatal("value field should be omitted")
	}
	if fieldByName(fields, "Links") != nil {
		t.Fatal("links field should be omitted")
	}
}

func TestBuildObservedValueFromAnnotationFallback(t *testing.T) {
	fields := buildFields(grafana.Alert{
		Status: "firing",
		Labels: map[string]string{
			"severity": "critical",
		},
		Annotations: map[string]string{
			"summary": "CPU saturation is high",
			"values":  "B=91.2, C=1",
		},
	}, "")

	observed := fieldByName(fields, "Value")
	if observed == nil || observed.Value != "91.2" {
		t.Fatalf("value field = %#v", observed)
	}
}

func TestBuildObservedValueFromGrafanaValueString(t *testing.T) {
	fields := buildFields(grafana.Alert{
		Status:      "resolved",
		Labels:      map[string]string{"severity": "warning"},
		Annotations: map[string]string{"summary": "Flux resource readiness is unknown"},
		ValueString: "[ var='B' labels={} value=0 ], [ var='C' labels={} value=0 ]",
	}, "")

	observed := fieldByName(fields, "Value")
	if observed == nil || observed.Value != "0" {
		t.Fatalf("value field = %#v", observed)
	}
}

func TestBuildObservedValueOmitsConditionOnlyValues(t *testing.T) {
	tests := []struct {
		name  string
		alert grafana.Alert
	}{
		{
			name: "native C only",
			alert: grafana.Alert{
				Status:      "firing",
				Labels:      map[string]string{"severity": "critical"},
				Annotations: map[string]string{"summary": "Threshold condition only"},
				Values:      map[string]float64{"C": 1},
			},
		},
		{
			name: "annotation C only",
			alert: grafana.Alert{
				Status:      "firing",
				Labels:      map[string]string{"severity": "critical"},
				Annotations: map[string]string{"summary": "Threshold condition only", "values": "C=1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := buildFields(tt.alert, "")
			if fieldByName(fields, "Value") != nil {
				t.Fatalf("value field should be omitted: %#v", fields)
			}
		})
	}
}

func TestBuildNextStepUsesRunbook(t *testing.T) {
	fields := buildFields(grafana.Alert{
		Status: "firing",
		Labels: map[string]string{
			"severity": "warning",
		},
		Annotations: map[string]string{
			"summary": "Discord alert delivery is failing",
			"runbook": "docs/runbooks/discord-alert-delivery-health.md",
		},
	}, "")

	nextStep := fieldByName(fields, "Next step")
	if nextStep == nil || nextStep.Value != "docs/runbooks/discord-alert-delivery-health.md" {
		t.Fatalf("next step field = %#v", nextStep)
	}
}

func TestFluxResolvedTriageCard(t *testing.T) {
	started := time.Date(2026, 7, 4, 12, 50, 0, 0, time.UTC)
	ended := time.Date(2026, 7, 4, 12, 55, 20, 0, time.UTC)
	msgs := GrafanaToDiscord(&grafana.WebhookPayload{
		Status:      "resolved",
		ExternalURL: "https://grafana.example.com",
		Alerts: []grafana.Alert{
			{
				Status: "resolved",
				Labels: map[string]string{
					"alertname": "Flux Resources Readiness Unknown",
					"component": "flux",
					"severity":  "warning",
				},
				Annotations: map[string]string{
					"summary":     "Flux resource readiness is unknown",
					"description": "Flux resource readiness is unknown [ var='B' labels={} value=0 ], [ var='C' labels={} value=0 ] Flux resource(s) report unknown readiness and may not be reconciling cleanly",
					"runbook":     "Start with: flux get all -A; kubectl describe kustomization -n <namespace> <name>; kubectl describe helmrelease -n <namespace> <name>",
				},
				ValueString:  "[ var='B' labels={} value=0 ], [ var='C' labels={} value=0 ]",
				GeneratorURL: "https://grafana.example.com/alerting/grafana/source",
				StartsAt:     started,
				EndsAt:       ended,
			},
		},
	})

	embed := msgs[0].Embeds[0]
	if embed.Title != "Recovered warning: Flux Resources Readiness Unknown" {
		t.Fatalf("title = %q", embed.Title)
	}
	if embed.Description != "Flux resource readiness is unknown" {
		t.Fatalf("description = %q", embed.Description)
	}
	value := fieldByName(embed.Fields, "Value")
	if value == nil || value.Value != "0" {
		t.Fatalf("value field = %#v", value)
	}
	timeline := fieldByName(embed.Fields, "Timeline")
	if timeline == nil || !strings.Contains(timeline.Value, "Duration: `5m20s`") {
		t.Fatalf("timeline field = %#v", timeline)
	}
	nextStep := fieldByName(embed.Fields, "Next step")
	if nextStep == nil || !strings.Contains(nextStep.Value, "flux get all -A") {
		t.Fatalf("next step field = %#v", nextStep)
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
