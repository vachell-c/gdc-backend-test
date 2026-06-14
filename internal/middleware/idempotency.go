package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/vasti/gdc-backend-test/internal/model"
	"github.com/vasti/gdc-backend-test/internal/repository"
)

// bodyRecorder wraps an http.ResponseWriter to capture the response body and
// status code for idempotency key storage. It replaces the original writer
// inside the Echo Response wrapper so that both the middleware and the upstream
// handler see the captured output.
type bodyRecorder struct {
	writer http.ResponseWriter
	body   bytes.Buffer
	status int
}

// Header returns the header map from the underlying writer.
func (r *bodyRecorder) Header() http.Header {
	return r.writer.Header()
}

// Write captures the data and also writes to the original response writer.
func (r *bodyRecorder) Write(b []byte) (int, error) {
	n, err := r.body.Write(b)
	if err != nil {
		return n, err
	}
	return r.writer.Write(b)
}

// WriteHeader captures the status code and delegates to the underlying writer.
func (r *bodyRecorder) WriteHeader(code int) {
	r.status = code
	r.writer.WriteHeader(code)
}

// IdempotencyMiddleware returns a middleware that enforces idempotency on
// POST /tasks requests using the Idempotency-Key header. If a stored response
// exists (within a 24-hour window), it is returned immediately. Otherwise the
// request proceeds and the response is stored for future lookups.
func IdempotencyMiddleware(idemRepo *repository.IdempotencyRepository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			// Only apply to POST methods on the /tasks path.
			if c.Request().Method != http.MethodPost || c.Request().URL.Path != "/tasks" {
				return next(c)
			}

			// Read and validate the Idempotency-Key header.
			idempotencyKeyStr := c.Request().Header.Get("Idempotency-Key")
			if idempotencyKeyStr == "" {
				return model.ErrValidation("Idempotency-Key header is required", nil)
			}

			key, err := uuid.Parse(idempotencyKeyStr)
			if err != nil {
				return model.ErrValidation("Idempotency-Key must be a valid UUID", err)
			}

			// Check if a stored response already exists.
			existing, err := idemRepo.GetByKey(c.Request().Context(), key)
			if err != nil {
				slog.ErrorContext(c.Request().Context(), "failed to check idempotency key",
					"key", key,
					"error", err,
				)
				return model.ErrInternal("failed to check idempotency key", err)
			}

			if existing != nil {
				// Return the stored response immediately.
				var body map[string]interface{}
				if err := json.Unmarshal(existing.ResponseBody, &body); err != nil {
					slog.ErrorContext(c.Request().Context(), "failed to unmarshal stored response body",
						"key", key,
						"error", err,
					)
					return c.JSONBlob(existing.ResponseCode, existing.ResponseBody)
				}
				return c.JSON(existing.ResponseCode, body)
			}

			// Install the body recorder by replacing the writer inside the
			// *echo.Response wrapper.
			echoResp, ok := c.Response().(*echo.Response)
			if !ok {
				// If the response is not an *echo.Response, fall through.
				slog.WarnContext(c.Request().Context(), "response is not *echo.Response, skipping idempotency capture")
				return next(c)
			}

			rec := &bodyRecorder{writer: echoResp.ResponseWriter}
			echoResp.ResponseWriter = rec

			// Default to 200 OK; the handler will override it via WriteHeader.
			c.Response().Header().Set(http.StatusText(http.StatusOK), "")

			err = next(c)

			// If the handler itself returned an error, we don't store the response.
			if err != nil {
				return err
			}

			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}

			// Only store successful (2xx) responses for idempotency.
			if status < 200 || status >= 400 {
				return nil
			}

			// Store the response for future idempotency lookups.
			bodyBytes := rec.body.Bytes()
			if len(bodyBytes) == 0 {
				bodyBytes = []byte("{}")
			}

			if saveErr := idemRepo.Save(c.Request().Context(), key, http.MethodPost, "/tasks", status, bodyBytes); saveErr != nil {
				slog.ErrorContext(c.Request().Context(), "failed to save idempotency key",
					"key", key,
					"error", saveErr,
				)
				// Non-fatal: the request succeeded, so we just log the storage failure.
			}

			return nil
		}
	}
}
