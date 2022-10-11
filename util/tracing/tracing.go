package tracing

import (
	"context"
	"net/http"
)

// Tracer represents an HTTP tracing facility.
type Tracer interface {
	NewTransport(base *http.Transport) http.RoundTripper
	StartSpan(ctx context.Context, name string) context.Context
	EndSpan(ctx context.Context, httpStatusCode int, err error)
}

var (
	tracer Tracer
)

// Register will register the provided Tracer.
func Register(t Tracer) {
	tracer = t
}

// IsEnabled returns true if a Tracer has been registered.
func IsEnabled() bool {
	return tracer != nil
}

// NewTransport creates a new instrumenting http.RoundTripper for the
// registered Tracer.  If no Tracer has been registered it returns nil.
func NewTransport(base *http.Transport) http.RoundTripper {
	if tracer != nil {
		return tracer.NewTransport(base)
	}
	return nil
}

// StartSpan starts a trace span with the specified name, associating it with the
// provided context.  Has no effect if a Tracer has not been registered.
func StartSpan(ctx context.Context, name string) context.Context {
	if tracer != nil {
		return tracer.StartSpan(ctx, name)
	}
	return ctx
}

// EndSpan ends a previously started span stored in the context.
// Has no effect if a Tracer has not been registered.
func EndSpan(ctx context.Context, httpStatusCode int, err error) {
	if tracer != nil {
		tracer.EndSpan(ctx, httpStatusCode, err)
	}
}
