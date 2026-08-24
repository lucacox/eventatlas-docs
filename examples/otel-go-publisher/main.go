package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const (
	defaultNATSURL          = "nats://127.0.0.1:4222"
	defaultServiceName      = "orders-go-publisher"
	defaultServiceNamespace = "eventatlas.examples"
	defaultEnvironment      = "development"
	defaultSubject          = "orders.created"
	defaultSubjectTemplate  = "orders.*"
	defaultMessage          = `{"event":"order.created","order_id":"demo-order-123"}`
	tracerName              = "github.com/lucacox/eventatlas-docs/examples/otel-go-publisher"
	exportShutdownTimeout   = 5 * time.Second
	natsOperationTimeout    = 5 * time.Second
)

type config struct {
	natsURL          string
	serviceName      string
	serviceNamespace string
	environment      string
	subject          string
	subjectTemplate  string
	message          string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() (runErr error) {
	ctx := context.Background()
	config := loadConfig()

	tracerProvider, err := newTracerProvider(ctx, config)
	if err != nil {
		return err
	}
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), exportShutdownTimeout)
		defer cancel()
		if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("shutdown OpenTelemetry provider: %w", err))
		}
	}()

	connection, err := nats.Connect(
		config.natsURL,
		nats.Name(config.serviceName),
		nats.Timeout(natsOperationTimeout),
	)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	defer connection.Close()

	tracer := otel.Tracer(tracerName)
	spanCtx, span := tracer.Start(
		ctx,
		config.subject+" send",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.operation.type", "send"),
			attribute.String("messaging.destination.name", config.subject),
			attribute.String("messaging.destination.template", config.subjectTemplate),
		),
	)

	publishErr := publish(spanCtx, connection, config.subject, []byte(config.message))
	if publishErr != nil {
		span.RecordError(publishErr)
		span.SetStatus(codes.Error, publishErr.Error())
	} else {
		span.SetStatus(codes.Ok, "message published")
	}
	span.End()
	if publishErr != nil {
		return publishErr
	}

	fmt.Printf(
		"published %q to %s as service %s; the span will be exported through OTLP/HTTP\n",
		config.message,
		config.subject,
		config.serviceName,
	)
	return nil
}

func newTracerProvider(ctx context.Context, config config) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create OTLP/HTTP trace exporter: %w", err)
	}

	serviceResource, err := resource.New(
		ctx,
		resource.WithAttributes(
			attribute.String("service.name", config.serviceName),
			attribute.String("service.namespace", config.serviceNamespace),
			attribute.String("deployment.environment.name", config.environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create OpenTelemetry resource: %w", err)
	}

	return sdktrace.NewTracerProvider(
		sdktrace.WithResource(serviceResource),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	), nil
}

func publish(ctx context.Context, connection *nats.Conn, subject string, payload []byte) error {
	if err := connection.Publish(subject, payload); err != nil {
		return fmt.Errorf("publish NATS message: %w", err)
	}

	flushCtx, cancel := context.WithTimeout(ctx, natsOperationTimeout)
	defer cancel()
	if err := connection.FlushWithContext(flushCtx); err != nil {
		return fmt.Errorf("flush NATS message: %w", err)
	}
	return nil
}

func loadConfig() config {
	return config{
		natsURL:          envOrDefault("NATS_URL", defaultNATSURL),
		serviceName:      envOrDefault("OTEL_SERVICE_NAME", defaultServiceName),
		serviceNamespace: envOrDefault("EXAMPLE_SERVICE_NAMESPACE", defaultServiceNamespace),
		environment:      envOrDefault("EXAMPLE_ENVIRONMENT", defaultEnvironment),
		subject:          envOrDefault("NATS_SUBJECT", defaultSubject),
		subjectTemplate:  envOrDefault("NATS_SUBJECT_TEMPLATE", defaultSubjectTemplate),
		message:          envOrDefault("MESSAGE_BODY", defaultMessage),
	}
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
