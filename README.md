# pretty-discord-alerts

Pretty Discord Alerts - A Go service that transforms Grafana webhooks into beautifully formatted Discord messages.

## Features

- 🎨 **Pretty Discord Embeds** - Transforms Grafana alerts into rich Discord embeds with colors, fields, and emojis
- 🚦 **Severity-Based Colors** - Critical (red), Warning (yellow), Resolved (green)
- 📊 **Alert Details** - Shows summary, description, namespace, and status for each alert
- 🔢 **Smart Limits** - Displays up to 10 alerts per message to avoid Discord embed limits
- ✅ **Health Probes** - Built-in health and readiness endpoints for Kubernetes

## Configuration

Set the following environment variables:

- `DISCORD_WEBHOOK_URL` (required) - Your Discord webhook URL
- `PORT` (optional) - Server port (default: 8888 locally, 8080 in Docker)

## Usage

### Running Locally

```bash
export DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/YOUR_WEBHOOK_URL"
go run main.go
```

Override the port:

```bash
PORT=3000 DISCORD_WEBHOOK_URL="..." go run main.go
```

### Building

```bash
go build -o pretty-discord-alerts
DISCORD_WEBHOOK_URL="..." ./pretty-discord-alerts
```

### Docker

Build the image:

```bash
docker build -t pretty-discord-alerts .
```

Run the container:

```bash
docker run -p 8080:8080 \
  -e DISCORD_WEBHOOK_URL="https://discord.com/api/webhooks/YOUR_WEBHOOK_URL" \
  pretty-discord-alerts
```

## Endpoints

- `POST /webhook` - Receives Grafana webhooks and forwards to Discord
- `GET /health` - Health check endpoint (returns `200` OK)
- `GET /ready` - Readiness probe for Kubernetes (returns `200` when ready, `503` when not ready)

## Grafana Setup

1. Get your Discord webhook URL from Discord Server Settings → Integrations → Webhooks
2. In Grafana, go to Alerting → Contact Points
3. Create a new contact point with type "Webhook"
4. Set the URL to: `http://your-service:8080/webhook`
5. Save and test!

## Testing

Send a test Grafana webhook:

```bash
curl -X POST http://localhost:8888/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "status": "firing",
    "alerts": [{
      "status": "firing",
      "labels": {
        "alertname": "HighCPUUsage",
        "severity": "critical",
        "namespace": "production"
      },
      "annotations": {
        "summary": "CPU usage is above 90%",
        "description": "The CPU usage has been above 90% for the last 5 minutes"
      }
    }]
  }'
```

## Discord Message Format

The service transforms Grafana alerts into Discord embeds with:

- **Title**: Emoji + alert status (🔥 Critical / ⚠️ Warning / ✅ Resolved)
- **Description**: Count of firing/resolved alerts
- **Fields**: Up to 10 alerts with details (summary, description, namespace, status)
- **Color**: Red (critical), Yellow (warning), Green (resolved)
- **Footer**: "Grafana Alerts" with Grafana icon
- **Timestamp**: Current time

## Health Checks

```bash
# Health check
curl -i http://localhost:8888/health

# Readiness check
curl -i http://localhost:8888/ready
```
