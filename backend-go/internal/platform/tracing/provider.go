package tracing

import (
	"context"
	"fmt"
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"gorm.io/gorm"
)

func InitTracerProvider(db *gorm.DB, cfg Config) (*sdktrace.TracerProvider, error) {
	exporter, err := NewSQLiteSpanExporter(db, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlite span exporter: %w", err)
	}

	res, err := sdkresource.New(
		context.Background(),
		sdkresource.WithAttributes(
			semconv.ServiceNameKey.String("rss-reader-backend"),
			semconv.ServiceVersionKey.String("1.0.0"),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var spanProcessors []sdktrace.SpanProcessor
	spanProcessors = append(spanProcessors, sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithBatchTimeout(5),
		sdktrace.WithMaxExportBatchSize(cfg.BufferSize),
	))

	if cfg.Debug {
		stdoutExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			log.Printf("[tracing] warning: failed to create stdout exporter: %v", err)
		} else {
			spanProcessors = append(spanProcessors, sdktrace.NewSimpleSpanProcessor(stdoutExporter))
		}
	}

	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
	}
	for _, sp := range spanProcessors {
		opts = append(opts, sdktrace.WithSpanProcessor(sp))
	}

	tp := sdktrace.NewTracerProvider(opts...)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Println("[tracing] TracerProvider initialized successfully")
	return tp, nil
}
