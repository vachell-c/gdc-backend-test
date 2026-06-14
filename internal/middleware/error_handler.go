package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/vasti/gdc-backend-test/internal/model"
)

// ErrorHandlerMiddleware returns a middleware that catches panics and converts
// handler errors to structured JSON error responses using model.FormatError
// and model.NewErrorResponse.
func ErrorHandlerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (err error) {
			// Recover from panics.
			defer func() {
				if r := recover(); r != nil {
					var ok bool
					err, ok = r.(error)
					if !ok {
						err = errors.New("panic recovered")
					}
					slog.ErrorContext(c.Request().Context(), "panic recovered",
						"panic", r,
						"error", err,
					)
					// Convert to internal error.
					appErr := model.ErrInternal("An internal error occurred", err)
					resp := model.NewErrorResponse(appErr)
					_ = c.JSON(appErr.Status, resp)
				}
			}()

			err = next(c)
			if err == nil {
				return nil
			}

			// Convert the error to an AppError and return JSON.
			appErr := model.FormatError(err)
			resp := model.NewErrorResponse(appErr)

			return c.JSON(appErr.Status, resp)
		}
	}
}

// GlobalErrorHandler is an echo.HTTPErrorHandler that converts all errors
// (including Echo's own HTTP errors and panics) into structured AppError JSON
// responses.
//
// Usage: e.HTTPErrorHandler = middleware.GlobalErrorHandler
func GlobalErrorHandler(c *echo.Context, err error) {
	// If the response has already been committed, just log and return.
	if resp, ok := c.Response().(*echo.Response); ok && resp.Committed {
		slog.ErrorContext(c.Request().Context(), "error after response committed",
			"error", err,
		)
		return
	}

	// Check for Echo's *echo.HTTPError and convert it.
	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		msg := httpErr.Message
		if msg == "" {
			msg = http.StatusText(httpErr.Code)
		}
		appErr := &model.AppError{
			Status:  httpErr.Code,
			Code:    http.StatusText(httpErr.Code),
			Message: msg,
			Err:     err,
		}
		resp := model.NewErrorResponse(appErr)
		if jsonErr := c.JSON(appErr.Status, resp); jsonErr != nil {
			slog.ErrorContext(c.Request().Context(), "failed to write error response",
				"original_error", err,
				"json_error", jsonErr,
			)
		}
		return
	}

	// Convert via model.FormatError for all other errors.
	appErr := model.FormatError(err)
	resp := model.NewErrorResponse(appErr)
	if jsonErr := c.JSON(appErr.Status, resp); jsonErr != nil {
		slog.ErrorContext(c.Request().Context(), "failed to write error response",
			"original_error", err,
			"json_error", jsonErr,
		)
	}
}
