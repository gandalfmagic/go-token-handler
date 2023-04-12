package opentelemetry

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey struct{}

func InitTracer(service, host, port string) (*sdktrace.TracerProvider, error) {
	// Create stdout exporter to be able to retrieve
	// the collected spans.
	exporter, err := jaeger.New(jaeger.WithAgentEndpoint(jaeger.WithAgentHost(host), jaeger.WithAgentPort(port)))
	if err != nil {
		return nil, err
	}

	// For the demonstration, use sdktrace.AlwaysSample sampler to sample all traces.
	// In a production application, use sdktrace.ProbabilitySampler with a desired probability.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceName(service))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, err
}

func NewContext(parent context.Context, t trace.Tracer) context.Context {
	return context.WithValue(parent, ctxKey{}, t)
}

func NewSpanFromContext(ctx context.Context, spanName string) (context.Context, trace.Span) {
	tracer, ok := ctx.Value(ctxKey{}).(trace.Tracer)
	if !ok {
		return ctx, nil
	}

	var span trace.Span
	if tracer != nil {
		ctx, span = tracer.Start(ctx, spanName)
	}

	return ctx, span
}

func TracerFromContext(ctx context.Context) trace.Tracer {
	t, _ := ctx.Value(ctxKey{}).(trace.Tracer)
	return t
}

func Middleware(next http.Handler, name, operation string) http.Handler {
	return otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := otel.GetTracerProvider().Tracer(name)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, tracer)))
	}), operation)
}
