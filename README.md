# pretty-discord-alerts

Pretty Discord Alerts - A Go service that transforms Grafana webhooks into beautifully formatted Discord messages.

## Features

- 🎨 **Pretty Discord Embeds** - Transforms Grafana alerts into clean, Datadog-inspired Discord embeds
- 🚦 **Severity-Based Colors** - Critical (red), Warning (orange/yellow), Resolved (green), Notification/Info (neutral gray-blue)
- 📊 **Alert Details** - Shows summary, scope labels, values, status timing, and actions for each alert
- 🔢 **Smart Limits** - Keeps field values within Discord embed limits
- ✅ **Health Probes** - Built-in health and readiness endpoints for Kubernetes

## Configuration

Set the following environment variables:

- `DISCORD_WEBHOOK_URL` (required) - Your Discord webhook URL
- `PORT` (optional) - Server port (default: 8888 locally, 8080 in Docker)
- `LOG_LEVEL` (optional) - Set log level: `debug`, `info`, `warn`, or `error` (default: `info`)
- `DEBUG` (optional) - Legacy option, equivalent to `LOG_LEVEL=debug` (set to `true`)

### Logging

The service uses structured JSON logging with the following levels:
- **INFO** (default): Success messages and important events
- **DEBUG**: Verbose logging including raw webhook request bodies
- **WARN**: Warning messages
- **ERROR**: Error messages and panics

Enable debug logging (either method works):
```bash
# Standard method
LOG_LEVEL=debug go run main.go

# Legacy method
DEBUG=true go run main.go
```

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
    "receiver": "discord",
    "status": "firing",
    "externalURL": "https://monitoring.example.com",
    "alerts": [{
      "status": "firing",
      "labels": {
        "alertname": "TestAlert",
        "grafana_folder": "Test Folder",
        "instance": "Grafana",
        "severity": "critical"
      },
      "annotations": {
        "summary": "Notification test",
        "values": "B=22, C=1"
      },
      "startsAt": "2026-02-02T12:00:00Z",
      "endsAt": "0001-01-01T00:00:00Z",
      "generatorURL": "https://monitoring.example.com/d/dashboard_uid?orgId=1",
      "fingerprint": "abc123"
    }],
    "groupLabels": {},
    "commonLabels": {
      "alertname": "TestAlert"
    },
    "commonAnnotations": {
      "summary": "Notification test"
    }
  }'
```

## Discord Message Format

The service sends **one Discord message per alert** with:

- **Username**: "Grafana"
- **Embed**:
  - **Title**: Datadog-inspired monitor status (`Critical monitor triggered`, `Warning monitor triggered`, `Monitor recovered`, or `Notification`)
  - **Description**: Alert name from the `alertname` label
  - **Fields**:
    - `Summary` with summary and description annotations
    - `Scope` with namespace, pod, instance, job, service, node, and component labels when present
    - `Observed value` from Grafana evaluation ref `B` when present
    - `Status` with firing/resolved state and start/end time when present
    - `Actions` with Source and Silence links
  - **Color**: Red for critical, orange/yellow for warning, green for resolved, neutral gray-blue for notification/info
  - **Type**: "rich"
  - **URL**: Link to Grafana alerting list
  - **Timestamp**: Alert start time for firing alerts, end time for resolved alerts
  - **Footer**: "Grafana monitor" with Grafana icon

### Example Discord Output

For a critical firing alert, each Discord message will look like:

**Username:** Grafana

**Embed Title:** Critical monitor triggered

**Description:** TestAlert

**Summary**
```
Notification test
```

**Observed value**
```
22
```

**Status**
```
Firing
Started: 2026-02-02T12:00:00Z
```

**Actions**
```
[Source](https://...) • [Silence](https://...)
```

**Footer:** Grafana monitor

> **Note**: 
> - Each alert in the Grafana payload creates a separate Discord message
> - "Observed value" shows Grafana evaluation ref `B` when present and omits threshold condition ref `C`

## Health Checks

```bash
# Health check
curl -i http://localhost:8888/health

# Readiness check
curl -i http://localhost:8888/ready
```
