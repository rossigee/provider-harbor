package tracing

import (
	"context"
	"os"
	"strconv"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracerName       = "provider-harbor"
	resourceTypeAttr = "crossplane.resource.type"
	resourceNameAttr = "crossplane.resource.name"
	operationAttr    = "crossplane.operation"
)

var tracer trace.Tracer
var tp *sdktrace.TracerProvider

func Init(serviceName string) func(context.Context) {
	tracer = otel.Tracer(tracerName)

	enabled, _ := strconv.ParseBool(getEnv("OTEL_TRACING_ENABLED", "false"))
	if !enabled {
		return func(context.Context) {}
	}

	endpoint := getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	samplingRatio := 0.1
	if v, err := strconv.ParseFloat(getEnv("OTEL_SAMPLING_RATIO", "0.1"), 64); err == nil {
		samplingRatio = v
	}

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(getEnv("OTEL_SERVICE_NAME", serviceName)),
			attribute.String("provider.type", "crossplane"),
		),
	)
	if err != nil {
		return func(context.Context) {}
	}

	exporter, err := otlptrace.New(ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		),
	)
	if err != nil {
		return func(context.Context) {}
	}

	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.TraceIDRatioBased(samplingRatio)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) {
		if tp != nil {
			_ = tp.Shutdown(ctx)
		}
	}
}

func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return tracer.Start(ctx, name,
		trace.WithAttributes(attrs...),
	)
}

func StartSpanWithAttrs(ctx context.Context, name, resourceType, resourceName, operation string) (context.Context, trace.Span) {
	return tracer.Start(ctx, name,
		trace.WithAttributes(
			attribute.String(resourceTypeAttr, resourceType),
			attribute.String(resourceNameAttr, resourceName),
			attribute.String(operationAttr, operation),
		),
	)
}

func SpanAttrs(resourceType, resourceName, operation string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String(resourceTypeAttr, resourceType),
		attribute.String(resourceNameAttr, resourceName),
		attribute.String(operationAttr, operation),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}