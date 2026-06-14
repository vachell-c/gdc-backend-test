package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// RequestIDMiddleware generates or forwards a request ID and stores it in the
// Echo context and response header.
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Use the existing X-Request-ID header, or generate a new UUID.
			requestID := c.Request().Header.Get(echo.HeaderXRequestID)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			c.Set("request_id", requestID)
			c.Response().Header().Set(echo.HeaderXRequestID, requestID)

			return next(c)
		}
	}
}
