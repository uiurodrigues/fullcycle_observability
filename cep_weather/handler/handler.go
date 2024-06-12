package handler

import (
	"cep_weather/configs"
	"cep_weather/dto"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/openzipkin/zipkin-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

var ErrCEPNotFound = fmt.Errorf("can not find zipcode")
var ErrCEPInvalid = fmt.Errorf("invalid zipcode")

type handler struct {
	tracer       trace.Tracer
	zipkinTracer *zipkin.Tracer
}

func NewHandler(zipkinTracer *zipkin.Tracer) handler {
	return handler{
		tracer:       otel.Tracer("cep-weather"),
		zipkinTracer: zipkinTracer,
	}
}

func (h *handler) GetWeatherHandler(w http.ResponseWriter, r *http.Request) {
	carrier := propagation.HeaderCarrier(r.Header)
	ctx := r.Context()
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)
	ctx, span := h.tracer.Start(ctx, "GetWeatherHandler")
	defer span.End()

	vars := mux.Vars(r)
	cep := vars["cep"]

	if !isCepValid(cep) {
		http.Error(w, ErrCEPInvalid.Error(), http.StatusUnprocessableEntity)
		return
	}

	location, err := h.getLocationByCEP(ctx, cep)
	if errors.Is(err, ErrCEPNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	weather, err := h.getWeatherByLocation(ctx, location.Location)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	weatherResponse := dto.NewCEPWeatherResponse(location, weather)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(weatherResponse)
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

func (h *handler) getLocationByCEP(ctx context.Context, cep string) (*dto.Location, error) {
	ctx, span := h.tracer.Start(ctx, "getCep")
	defer span.End()

	zipkinSpan, _ := h.zipkinTracer.StartSpanFromContext(ctx, "getCep")
	defer zipkinSpan.Finish()

	url := fmt.Sprintf("http://viacep.com.br/ws/%s/json/", cep)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Printf("error creating ViaCEP request. Err:%s", err.Error())
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("error executing ViaCEP request. Err:%s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {

	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("error while reading ViaCEP result. Err:%s", err.Error())
			return nil, err
		}

		var location *dto.Location
		if err = json.Unmarshal(body, &location); err != nil {
			return nil, err
		}
		if location.CEP == "" {
			return nil, ErrCEPNotFound
		}
		return location, nil

	case http.StatusNotFound:
		return nil, ErrCEPNotFound

	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func (h *handler) getWeatherByLocation(ctx context.Context, location string) (*dto.Weather, error) {
	ctx, span := h.tracer.Start(ctx, "getWeather")
	defer span.End()

	zipkinSpan, _ := h.zipkinTracer.StartSpanFromContext(ctx, "getWeather")
	defer zipkinSpan.Finish()

	location = strings.Replace(location, " ", "%20", -1) //threat space in location
	reqUrl := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s", configs.GetConfig().WeatherAPIKey, url.PathEscape(location))

	req, err := http.NewRequest(http.MethodGet, reqUrl, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		log.Printf("error creating weatherAPI request. Err:%s", err.Error())
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("error executing weatherAPI request. Err:%s", err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("error while getting weatherAPI result. Status: %s, Body: %s", resp.Status, string(body))

		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("error while reading weatherAPI result. Err:%s", err.Error())
		return nil, err
	}

	var weather *dto.Weather
	if err = json.Unmarshal(body, &weather); err != nil {
		log.Printf("error while converting weatherAPI result. Err:%s", err.Error())
		return nil, err
	}
	return weather, nil
}
