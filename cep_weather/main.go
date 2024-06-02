package main

import (
	"cep_weather/handler"
	"cep_weather/middleware"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	log.Println("Starting server...")
	defer log.Println("Server finished...")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	shutdown, err := initProvider(ctx, "cep-weather", "otel-collector:4317")
	if err != nil {
		log.Fatalf("error initializing provider: %v", err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Fatalf("error shutting down provider: %v", err)
		}
	}()

	go func() {
		weatherHandler := handler.NewHandler()

		zipkinTracer, err := middleware.NewZipkinTracer()
		if err != nil {
			log.Fatalf("failed to create zipkin tracer: %s", err.Error())
		}
		http.DefaultClient.Transport, err = zipkinhttp.NewTransport(
			zipkinTracer,
			zipkinhttp.TransportTrace(true),
		)

		r := mux.NewRouter()
		r.HandleFunc("/weather/{cep}", weatherHandler.GetWeatherHandler).Methods(http.MethodGet)
		r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
		r.Use(zipkinhttp.NewServerMiddleware(
			zipkinTracer,
			zipkinhttp.SpanName("request_cep_weather")),
		)

		log.Println("listening on port 8080")
		http.ListenAndServe(":8080", r)
	}()

	select {
	case <-sigCh:
		log.Println("Shutting down gracefully, CTRL+C pressed")
	case <-ctx.Done():
		log.Println("Shutting down due to other reason...")
	}

	_, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
}

func initProvider(ctx context.Context, serviceName, collectorURL string) (func(context.Context) error, error) {
	res, err := resource.New(
		ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, collectorURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to colletor: %w", err)
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(traceProvider)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	return traceProvider.Shutdown, nil
}
