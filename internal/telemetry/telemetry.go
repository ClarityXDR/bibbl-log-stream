package telemetry

import (
	"context"
	"fmt"
	"strings"

	"bibbl/internal/config"
	"bibbl/internal/version"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Init configures the OTLP exporter when an endpoint is provided. It returns a shutdown function.
func Init(ctx context.Context, cfg config.TelemetryConfig) (func(context.Context) error, error) {
	endpoint := strings.TrimSpace(cfg.OTLP.Endpoint)
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}
	if cfg.OTLP.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	if cfg.OTLP.Timeout > 0 {
		opts = append(opts, otlptracegrpc.WithTimeout(cfg.OTLP.Timeout))
	}
	if cfg.OTLP.Compression != "" {
		opts = append(opts, otlptracegrpc.WithCompressor(strings.ToLower(cfg.OTLP.Compression)))
	}
	if len(cfg.OTLP.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(cfg.OTLP.Headers))
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("init otlp exporter: %w", err)
	}

	ratio := cfg.OTLP.SampleRatio
	if ratio <= 0 || ratio > 1 {
		ratio = 1
	}
	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("bibbl-log-stream"),
			semconv.ServiceVersionKey.String(version.Version),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("init otlp resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
