// Package telemetry provides opt-in OpenTelemetry traces and metrics without a collector.
package telemetry

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

// Runtime owns the providers used to instrument HTTP requests.
type Runtime struct {
	tracer   trace.Tracer                `json:"-"`
	requests metric.Int64Counter         `json:"-"`
	duration metric.Float64Histogram     `json:"-"`
	shutdown func(context.Context) error `json:"-"`
}

// New configures stdout exporters when enabled, otherwise using no-op providers.
// The caller must call Shutdown after stopping HTTP request handling.
func New(enabled bool, output io.Writer) (*Runtime, error) {
	var (
		tracerProvider trace.TracerProvider = tracenoop.NewTracerProvider()
		meterProvider  metric.MeterProvider = metricnoop.NewMeterProvider()
		shutdown                            = func(context.Context) error { return nil }
	)
	if enabled {
		traceExporter, err := stdouttrace.New(stdouttrace.WithWriter(output))
		if err != nil {
			return nil, err
		}
		metricExporter, err := stdoutmetric.New(stdoutmetric.WithWriter(output))
		if err != nil {
			return nil, errors.Join(err, traceExporter.Shutdown(context.Background()))
		}
		traces := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter))
		meters := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)))
		tracerProvider, meterProvider = traces, meters
		shutdown = func(ctx context.Context) error {
			return errors.Join(traces.Shutdown(ctx), meters.Shutdown(ctx))
		}
	}
	meter := meterProvider.Meter("go-practices")
	requests, err := meter.Int64Counter("http.server.requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, errors.Join(err, shutdown(context.Background()))
	}
	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, errors.Join(err, shutdown(context.Background()))
	}
	return &Runtime{tracer: tracerProvider.Tracer("go-practices"), requests: requests, duration: duration, shutdown: shutdown}, nil
}

// Shutdown flushes and closes both providers using the caller's deadline.
func (r *Runtime) Shutdown(ctx context.Context) error { return r.shutdown(ctx) }

// Wrap records request spans, counts, and duration without URLs, headers, or claims.
func (r *Runtime) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		start := time.Now()
		ctx, span := r.tracer.Start(request.Context(), "HTTP request", trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()
		// Do not include arbitrary method strings as metric attributes.
		method := request.Method
		switch method {
		case "GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH":
		default:
			method = "_OTHER"
		}
		attrs := metric.WithAttributes(attribute.String("http.request.method", method))
		defer func() {
			r.requests.Add(ctx, 1, attrs)
			r.duration.Record(ctx, time.Since(start).Seconds(), attrs)
		}()
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}
