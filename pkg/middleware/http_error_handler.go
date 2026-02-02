package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/pretty-discord-alerts/pkg/metrics"
)

type HTTPError struct {
	Status  int
	Message string
	Cause   error
}

func (e *HTTPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func RecoverMiddleware(next http.HandlerFunc, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() {
			if rec := recover(); rec != nil {
				var status int
				var message string
				var metricLabel string

				switch e := rec.(type) {
				case *HTTPError:
					status = e.Status
					message = e.Message
					slog.Error("HTTP error", "status", status, "path", path, "error", e)
					if status >= 500 {
						metricLabel = "discord_error"
					} else {
						metricLabel = "decode_error"
					}
				case error:
					status = http.StatusInternalServerError
					message = "Internal server error"
					metricLabel = "panic"
					slog.Error("Panic recovered", "path", path, "error", e)
				default:
					status = http.StatusInternalServerError
					message = "Internal server error"
					metricLabel = "panic"
					slog.Error("Panic recovered", "path", path, "error", rec)
				}

				metrics.RecordHTTPRequest(path, r.Method, strconv.Itoa(status), time.Since(start))
				if path == "/webhook" {
					metrics.RecordWebhookRequest(metricLabel)
				}
				http.Error(w, message, status)
			}
		}()
		next(w, r)
	}
}