package tele

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracer struct {
	trace.Tracer
}

// Start tracer
func (t tracer) Start(
	ctx context.Context,
	op string,
	opts ...trace.SpanStartOption,
) (context.Context, trace.Span) {
	ctx, corrID := ctxWithCorrID(ctx)
	opts = append(
		opts,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(attribute.String(
			string(CorrIDKeyVal),
			string(corrID),
		)),
	)
	return t.Tracer.Start(ctx, op, opts...)
}

// Return tracer
func Tracer() trace.Tracer {
	return tracer{
		Tracer: otel.Tracer("capo"),
	}
}
