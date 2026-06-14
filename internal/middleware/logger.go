package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"
)

// LoggerMiddleware returns a middleware that logs each HTTP request with
// structured JSON via slog, including request_id, method, path, status,
// latency, and log level based on the response status code.
func LoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			start := time.Now()

			// Extract request_id from context (set by RequestIDMiddleware).
			requestID, _ := c.Get("request_id").(string)

			// Process the request.
			err := next(c)

			latency := time.Since(start)
			latencyStr := fmt.Sprintf("%.3fms", float64(latency.Microseconds())/1000.0)

			// Obtain the response status via type assertion on the underlying
			// *echo.Response wrapper.
			resp, _ := c.Response().(*echo.Response)
			status := 0
			if resp != nil {
				status = resp.Status
			}

			method := c.Request().Method
			path := c.Request().URL.Path

			// Create a context with the request_id for structured logging.
			logCtx := slog.With("request_id", requestID)

			switch {
			case status >= 500:
				logCtx.ErrorContext(c.Request().Context(), "request completed",
					"method", method,
					"path", path,
					"status", status,
					"latency", latencyStr,
				)
			case status >= 400:
				logCtx.WarnContext(c.Request().Context(), "request completed",
					"method", method,
					"path", path,
					"status", status,
					"latency", latencyStr,
				)
			default:
				logCtx.InfoContext(c.Request().Context(), "request completed",
					"method", method,
					"path", path,
					"status", status,
					"latency", latencyStr,
				)
			}

			return err
		}
	}
}
