/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
