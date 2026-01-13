package cmd

import (
	"context"
	"log"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// =====================================================
// OPENTELEMETRY TRACING ДЛЯ KBOT
// =====================================================
// Цей файл додає distributed tracing з наскрізним TraceID
// Traces експортуються до OTEL Collector → Jaeger
// =====================================================

var tracer trace.Tracer

// InitTracer ініціалізує OpenTelemetry tracer
func InitTracer(ctx context.Context) (func(context.Context) error, error) {
	// Отримуємо endpoint з env або використовуємо default
	otelEndpoint := getEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector-collector.monitoring.svc.cluster.local:4317")

	// Створюємо OTLP exporter (gRPC)
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(), // Для dev середовища без TLS
	)
	if err != nil {
		return nil, err
	}

	// Створюємо resource з інформацією про сервіс
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("kbot"),
			semconv.ServiceVersion(appVersion),
			attribute.String("environment", getEnvOrDefault("ENVIRONMENT", "development")),
		),
	)
	if err != nil {
		return nil, err
	}

	// Створюємо TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // Для dev - семплюємо все
	)

	// Встановлюємо глобальний TracerProvider
	otel.SetTracerProvider(tp)

	// Встановлюємо propagator для передачі контексту між сервісами
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Створюємо tracer для використання в коді
	tracer = tp.Tracer("kbot")

	log.Printf("OpenTelemetry tracing initialized, exporting to: %s", otelEndpoint)

	// Повертаємо функцію shutdown
	return tp.Shutdown, nil
}

// StartSpan створює новий span для операції
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		// Якщо tracer не ініціалізовано, повертаємо noop span
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, name, opts...)
}

// GetTraceID повертає TraceID з контексту (для логування)
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// AddSpanAttributes додає атрибути до поточного span
func AddSpanAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordSpanError записує помилку в span
func RecordSpanError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// IsTracingEnabled перевіряє чи увімкнено трейсинг
func IsTracingEnabled() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_TRACING_ENABLED") == "true"
}
