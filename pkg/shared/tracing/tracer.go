// Package tracing provides OpenTelemetry distributed tracing utilities.
// It enables tracing across all microservices for request flow visibility.
package tracing

import (
	"context"
	"fmt"
	"time"

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

// Config holds tracing configuration.
type Config struct {
	ServiceName    string  `json:"service_name"`
	ServiceVersion string  `json:"service_version"`
	Environment    string  `json:"environment"`
	Endpoint       string  `json:"endpoint"`
	SampleRate     float64 `json:"sample_rate"`
	Enabled        bool    `json:"enabled"`
}

// DefaultConfig returns default tracing configuration.
func DefaultConfig(serviceName string) *Config {
	return &Config{
		ServiceName:    serviceName,
		ServiceVersion: "1.0.0",
		Environment:    "development",
		Endpoint:       "localhost:4317",
		SampleRate:     1.0,
		Enabled:        true,
	}
}

// Tracer wraps the OpenTelemetry tracer with additional functionality.
type Tracer struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	config   *Config
}

// Provider interface for getting trace provider.
type Provider interface {
	Shutdown(ctx context.Context) error
	Tracer() trace.Tracer
}

// NewTracer creates a new Tracer instance.
func NewTracer(cfg *Config) (*Tracer, error) {
	if !cfg.Enabled {
		return &Tracer{
			tracer: otel.Tracer(cfg.ServiceName),
			config: cfg,
		}, nil
	}

	ctx := context.Background()

	// Create OTLP exporter
	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(),
	)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create resource with service information
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			attribute.String("environment", cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create sampler based on sample rate
	var sampler sdktrace.Sampler
	if cfg.SampleRate >= 1.0 {
		sampler = sdktrace.AlwaysSample()
	} else if cfg.SampleRate <= 0 {
		sampler = sdktrace.NeverSample()
	} else {
		sampler = sdktrace.TraceIDRatioBased(cfg.SampleRate)
	}

	// Create trace provider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Set global trace provider and propagator
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Tracer{
		tracer:   provider.Tracer(cfg.ServiceName),
		provider: provider,
		config:   cfg,
	}, nil
}

// Tracer returns the underlying OpenTelemetry tracer.
func (t *Tracer) Tracer() trace.Tracer {
	return t.tracer
}

// Shutdown gracefully shuts down the tracer provider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	if t.provider != nil {
		return t.provider.Shutdown(ctx)
	}
	return nil
}

// StartSpan starts a new span with the given name.
func (t *Tracer) StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// StartSpanFromContext starts a span from the given context.
func StartSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return otel.Tracer("").Start(ctx, name, opts...)
}

// SpanFromContext returns the current span from context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// AddEvent adds an event to the current span.
func AddEvent(ctx context.Context, name string, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.AddEvent(name, trace.WithAttributes(attrs...))
}

// SetAttributes sets attributes on the current span.
func SetAttributes(ctx context.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attrs...)
}

// RecordError records an error on the current span.
func RecordError(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
}

// GetTraceID returns the trace ID from the current span context.
func GetTraceID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasTraceID() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// GetSpanID returns the span ID from the current span context.
func GetSpanID(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().HasSpanID() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// Attribute helpers for common attributes.

// ServiceAttr creates a service name attribute.
func ServiceAttr(name string) attribute.KeyValue {
	return semconv.ServiceName(name)
}

// HTTPMethodAttr creates an HTTP method attribute.
func HTTPMethodAttr(method string) attribute.KeyValue {
	return semconv.HTTPMethod(method)
}

// HTTPURLAttr creates an HTTP URL attribute.
func HTTPURLAttr(url string) attribute.KeyValue {
	return semconv.HTTPURL(url)
}

// HTTPStatusCodeAttr creates an HTTP status code attribute.
func HTTPStatusCodeAttr(code int) attribute.KeyValue {
	return semconv.HTTPStatusCode(code)
}

// ErrorAttr creates an error attribute.
func ErrorAttr(err error) attribute.KeyValue {
	return attribute.String("error.message", err.Error())
}

// StringAttr creates a string attribute.
func StringAttr(key, value string) attribute.KeyValue {
	return attribute.String(key, value)
}

// IntAttr creates an int attribute.
func IntAttr(key string, value int) attribute.KeyValue {
	return attribute.Int(key, value)
}

// Int64Attr creates an int64 attribute.
func Int64Attr(key string, value int64) attribute.KeyValue {
	return attribute.Int64(key, value)
}

// Float64Attr creates a float64 attribute.
func Float64Attr(key string, value float64) attribute.KeyValue {
	return attribute.Float64(key, value)
}

// BoolAttr creates a bool attribute.
func BoolAttr(key string, value bool) attribute.KeyValue {
	return attribute.Bool(key, value)
}
