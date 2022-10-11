package opentracing

import (
	"fmt"
	crprometheus "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

// RegisterMetrics enables prometheus metrics for OpenTelemetry.
func RegisterMetrics() error {
	fmt.Printf("Register metrics opentracing \n")
	config := prometheus.Config{
		Registry: metrics.Registry.(*crprometheus.Registry), // use the controller runtime metrics registry / gatherer
	}
	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
	)
	exporter, err := prometheus.New(config, c)
	if err != nil {
		return err
	}
	global.SetMeterProvider(exporter.MeterProvider())

	return nil
}
