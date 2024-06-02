# CEP Weather

System made for finding the weather of an specific CEP.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Traces](#traces)

## Installation

1. Generate one API Key from WeatherAPI website and add it on the ./cep_weather.env file on the root of this application
2. Execute the command [ docker-compose  up -d ] to run the application on docker

## Usage

1. Execute an Get request using the following curl:
        curl --location 'http://localhost:8081' \
        --header 'Content-Type: application/json' \
        --data '{
            "cep":"00000000"
        }'

## Traces

1. To access zipkins: [http://localhost:9411/zipkin/]
2. To access jaeger: [http://localhost:16686/search]
