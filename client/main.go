package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/gorilla/mux"
	zipkinhttp "github.com/openzipkin/zipkin-go/middleware/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wirodrigues_meli/fullcycle_observability/client/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ErrCEPNotFound = fmt.Errorf("can not find zipcode")
var ErrCEPInvalid = fmt.Errorf("invalid zipcode")
var ErrInternalServerError = fmt.Errorf("internal server error")
var tracer trace.Tracer

func main() {
	log.Println("Starting client...")
	defer log.Println("Client finished...")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("cep_client"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %s", err.Error())
	}

	ctx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "otel-collector:4317",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("failed to create gRPC connection to collector: %s", err.Error())
	}

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		log.Fatalf("failed to create trace exporter: %s", err.Error())
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	tracer = otel.Tracer("client-cep")
	zipkinTracer, err := middleware.NewZipkinTracer()
	if err != nil {
		log.Fatalf("failed to create zipkin tracer: %s", err.Error())
	}

	http.DefaultClient.Transport, err = zipkinhttp.NewTransport(
		zipkinTracer,
		zipkinhttp.TransportTrace(true),
	)

	r := mux.NewRouter()
	r.HandleFunc("/", postToCepWeatherHandler).Methods(http.MethodPost)
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	r.Use(zipkinhttp.NewServerMiddleware(
		zipkinTracer,
		zipkinhttp.SpanName("request_client_cep")),
	)

	log.Println("client is listening on port 8081")
	http.ListenAndServe(":8081", r)

	select {
	case <-sigCh:
		log.Println("Shutting down gracefully, CTRL+C pressed...")
	case <-ctx.Done():
		log.Println("Shutting down due to other reason...")
	}
}

func postToCepWeatherHandler(w http.ResponseWriter, r *http.Request) {
	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := tracer.Start(ctx, "postToCepWeatherHandler")
	defer span.End()

	receivedReq := cepRequest{}
	if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !isCepValid(receivedReq.CEP) {
		fmt.Printf("invalid cep: %s", receivedReq.CEP)
		http.Error(w, ErrCEPInvalid.Error(), http.StatusUnprocessableEntity)
		return
	}

	url := fmt.Sprintf("http://cep_weather:8080/weather/%s", receivedReq.CEP)
	cepWeatherReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Printf("error while creating request: %s", err)
		http.Error(w, ErrInternalServerError.Error(), http.StatusInternalServerError)
		return
	}

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(cepWeatherReq.Header))
	resp, err := http.DefaultClient.Do(cepWeatherReq)
	if err != nil {
		log.Printf("error while making request: %s", err)
		http.Error(w, ErrInternalServerError.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, ErrInternalServerError.Error(), http.StatusInternalServerError)
			return
		}
		var location *CEPWeatherResponse
		if err = json.Unmarshal(body, &location); err != nil {
			log.Printf("error while unmarshaling response: %s", err)
			http.Error(w, ErrInternalServerError.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
		return

	case http.StatusNotFound:
		log.Printf("error while making request: %s", err)
		http.Error(w, ErrCEPNotFound.Error(), http.StatusNotFound)
		return

	default:
		log.Printf("unexpected error: %s", err)
		http.Error(w, ErrInternalServerError.Error(), http.StatusInternalServerError)
		return
	}

}

type cepRequest struct {
	CEP string `json:"cep"`
}

func isCepValid(cep string) bool {
	if cep == "" {
		return false
	}
	if len(cep) != 8 {
		return false
	}
	if !regexp.MustCompile(`^[0-9]*$`).MatchString(cep) {
		return false
	}
	return true
}

type CEPWeatherResponse struct {
	Location                string  `json:"city"`
	TemperatureInCelcius    float64 `json:"temp_C"`
	TemperatureInFahrenheit float64 `json:"temp_F"`
	TemperatureInKelvin     float64 `json:"temp_K"`
}
