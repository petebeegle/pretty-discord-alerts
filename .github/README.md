# GitHub Actions CI/CD

This project uses GitHub Actions for continuous integration and deployment.

## Workflows

### CI/CD Pipeline (`.github/workflows/ci.yml`)

Runs on:
- Push to `main` branch
- Pull requests to `main`
- Git tags starting with `v` (e.g., `v1.0.0`)

**Jobs:**

1. **Lint** - Runs `golangci-lint` to check code quality
2. **Test** - Runs all tests with race detection and coverage reporting
3. **Build and Push** - Builds multi-platform Docker images and pushes to GitHub Container Registry (ghcr.io)

## Docker Images

Docker images are automatically published to GitHub Container Registry:
- **Registry**: `ghcr.io/<owner>/pretty-discord-alerts`
- **Platforms**: `linux/amd64`, `linux/arm64`

### Image Tags

- `latest` - Latest build from main branch
- `main` - Latest build from main branch
- `v*` - Semantic version tags (e.g., `v1.0.0`, `v1.0`, `v1`)
- `main-<sha>` - Commit SHA tags

### Usage

```bash
# Pull the latest image
docker pull ghcr.io/<owner>/pretty-discord-alerts:latest

# Run the container
docker run -d \
  -p 8080:8080 \
  -e DISCORD_WEBHOOK_URL="your-webhook-url" \
  ghcr.io/<owner>/pretty-discord-alerts:latest
```

## Local Development

Run linting locally:
```bash
golangci-lint run
```

Run tests:
```bash
go test -v ./...
```

Run tests with coverage:
```bash
go test -v -race -coverprofile=coverage.txt ./...
```

Build Docker image locally:
```bash
docker build -t pretty-discord-alerts .
```

## Releasing

To create a new release:

1. Create and push a tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically:
   - Run lint and tests
   - Build multi-platform Docker images
   - Push images to ghcr.io with version tags

## Secrets

The workflow uses the following secrets:
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions for pushing to ghcr.io
- `CODECOV_TOKEN` - Optional, for uploading coverage to Codecov
