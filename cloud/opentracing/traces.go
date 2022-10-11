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

package opentracing

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/tele"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/util/tracing"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"time"
)

// RegisterTracing enables code tracing via OpenTelemetry.
func RegisterTracing(ctx context.Context, log logr.Logger) error {
	tp, err := otlpTracerProvider(ctx, "opentelemetry-collector:4317")
	if err != nil {
		return err
	}
	fmt.Println("Begin registering opentelemetry")
	otel.SetTracerProvider(tp)
	tracing.Register(NewOpenTelemetryAutorestTracer(tele.Tracer()))

	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Error(err, "failed to shut down tracer provider")
		}
	}()

	return nil
}

// otlpTracerProvider initializes an OTLP exporter and configures the corresponding tracer provider.
func otlpTracerProvider(ctx context.Context, url string) (*sdktrace.TracerProvider, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("capo"),
			attribute.String("exporter", "otlp"),
		),
	)
	fmt.Printf("Set Tracer Provider")

	if err != nil {
		return nil, errors.Wrap(err, "failed to create opentelemetry resource")
	}

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(url),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create otlp trace exporter")
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tracerProvider, nil
}
