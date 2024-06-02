package middleware

import (
	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	reporterhttp "github.com/openzipkin/zipkin-go/reporter/http"
)

const endpointURL = "http://zipkin:9411/api/v2/spans"

func NewZipkinTracer() (*zipkin.Tracer, error) {
	reporter := reporterhttp.NewReporter(endpointURL)

	localEndpoint := &model.Endpoint{ServiceName: "cep-weather", Port: 8080}

	sampler, err := zipkin.NewCountingSampler(1)
	if err != nil {
		return nil, err
	}

	t, err := zipkin.NewTracer(
		reporter,
		zipkin.WithSampler(sampler),
		zipkin.WithLocalEndpoint(localEndpoint),
	)
	if err != nil {
		return nil, err
	}

	return t, err
}
