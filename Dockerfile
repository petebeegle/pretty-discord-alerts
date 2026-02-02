# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pretty-discord-alerts .

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1001 appgroup && \
    adduser -D -u 1001 -G appgroup appuser

WORKDIR /home/appuser

COPY --from=builder /app/pretty-discord-alerts .

RUN chown -R appuser:appgroup /home/appuser
USER appuser

ENV PORT=8080
ENV DISCORD_WEBHOOK_URL=""
EXPOSE 8080

CMD ["./pretty-discord-alerts"]
