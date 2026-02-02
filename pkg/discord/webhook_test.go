package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewWebhook(t *testing.T) {
	url := "https://discord.com/api/webhooks/123/abc"
	webhook := NewWebhook(url)

	if webhook.URL != url {
		t.Errorf("NewWebhook() URL = %q, want %q", webhook.URL, url)
	}
}

func TestWebhook_Send_Success(t *testing.T) {
	// Create a test server that simulates Discord webhook endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify content type
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", ct)
		}

		// Verify request body can be decoded
		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Discord returns 204 No Content on success
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	webhook := NewWebhook(server.URL)
	msg := Message{
		Content: "Test message",
		Embeds: []Embed{
			{
				Title:       "Test Embed",
				Description: "Test Description",
				Color:       15158332,
			},
		},
	}

	err := webhook.Send(msg)
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}
}

func TestWebhook_Send_InvalidStatusCode(t *testing.T) {
	// Create a test server that returns a non-204 status code
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	webhook := NewWebhook(server.URL)
	msg := Message{Content: "Test"}

	err := webhook.Send(msg)
	if err == nil {
		t.Error("Send() error = nil, want error")
	}

	expectedErr := "unexpected status code: 400"
	if err.Error() != expectedErr {
		t.Errorf("Send() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestWebhook_Send_InvalidURL(t *testing.T) {
	webhook := NewWebhook("http://invalid-url-that-does-not-exist.local:99999")
	msg := Message{Content: "Test"}

	err := webhook.Send(msg)
	if err == nil {
		t.Error("Send() error = nil, want error for invalid URL")
	}
}

func TestWebhook_Send_MarshalError(t *testing.T) {
	// This is difficult to trigger with the current Message struct
	// since json.Marshal handles most types. We'll skip this for now
	// as it would require unsafe reflection or invalid types.
	t.Skip("Marshal error is difficult to trigger with current implementation")
}

func TestMessage_JSON(t *testing.T) {
	msg := Message{
		Content: "Test content",
		Embeds: []Embed{
			{
				Title:       "Title",
				Description: "Description",
				Color:       15158332,
				Fields: []EmbedField{
					{
						Name:   "Field 1",
						Value:  "Value 1",
						Inline: true,
					},
				},
				Footer: &EmbedFooter{
					Text:    "Footer text",
					IconURL: "https://example.com/icon.png",
				},
				Timestamp: "2026-02-02T12:00:00Z",
			},
		},
	}

	// Verify it can be marshaled
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Verify it can be unmarshaled
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal message: %v", err)
	}

	// Verify key fields
	if decoded.Content != msg.Content {
		t.Errorf("Content = %q, want %q", decoded.Content, msg.Content)
	}

	if len(decoded.Embeds) != 1 {
		t.Fatalf("Embeds length = %d, want 1", len(decoded.Embeds))
	}

	embed := decoded.Embeds[0]
	if embed.Title != "Title" {
		t.Errorf("Embed.Title = %q, want %q", embed.Title, "Title")
	}

	if embed.Color != 15158332 {
		t.Errorf("Embed.Color = %d, want %d", embed.Color, 15158332)
	}

	if len(embed.Fields) != 1 {
		t.Fatalf("Fields length = %d, want 1", len(embed.Fields))
	}

	if embed.Footer == nil {
		t.Fatal("Footer is nil")
	}

	if embed.Footer.Text != "Footer text" {
		t.Errorf("Footer.Text = %q, want %q", embed.Footer.Text, "Footer text")
	}
}

func TestEmbed_OmitEmpty(t *testing.T) {
	// Test that omitempty works correctly
	embed := Embed{
		Title: "Only Title",
	}

	data, err := json.Marshal(embed)
	if err != nil {
		t.Fatalf("Failed to marshal embed: %v", err)
	}

	// Verify empty fields are omitted
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := decoded["description"]; exists {
		t.Error("Expected description to be omitted, but it exists")
	}

	if _, exists := decoded["color"]; exists {
		t.Error("Expected color to be omitted when zero, but it exists")
	}

	if val, exists := decoded["title"]; !exists || val != "Only Title" {
		t.Error("Expected title to be present")
	}
}

func TestEmbedField_Inline(t *testing.T) {
	field := EmbedField{
		Name:   "Test",
		Value:  "Value",
		Inline: true,
	}

	data, err := json.Marshal(field)
	if err != nil {
		t.Fatalf("Failed to marshal field: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if inline, ok := decoded["inline"].(bool); !ok || !inline {
		t.Error("Expected inline to be true")
	}
}
