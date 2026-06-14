package middleware

import (
	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TraceMiddleware returns an Echo middleware that creates an OpenTelemetry span
// for each incoming request, attaching the request_id, method, path, and status code
// as span attributes.
func TraceMiddleware() echo.MiddlewareFunc {
	tracer := otel.Tracer("gdc-task-api")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			ctx := req.Context()

			spanName := req.Method + " " + req.URL.Path
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			span.SetAttributes(
				attribute.String("http.method", req.Method),
				attribute.String("http.path", req.URL.Path),
				attribute.String("http.host", req.Host),
			)

			if rid, ok := c.Get("request_id").(string); ok {
				span.SetAttributes(attribute.String("request_id", rid))
			}

			c.SetRequest(req.WithContext(ctx))

			err := next(c)

			// Read the response status from Echo's Response if available
			if echoResp, ok := c.Response().(*echo.Response); ok {
				span.SetAttributes(attribute.Int("http.status_code", echoResp.Status))
			}

			return err
		}
	}
}
