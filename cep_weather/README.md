# CEP Weather

System made for finding the weather of an specific CEP.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Tests](#tests)

## Installation

1. Generate one API Key from WeatherAPI website and add it on the .env file on the root of this application
2. Execute the command [ docker build -t cep_weather . ] to build the docker image of the application
3. Execute the command [ docker-compose  up -d ] to run the application on docker

## Usage

1. Execute an Get request on the address htt://localhost:8080/weather/{cep}, replacing the {cep} for one valid CEP
2. It will returns:
    - 200: The request is valid and the response should be shown
    - 404: CEP not found
    - 422: CEP is Invalid
3. To use it without execute in localhost, just access the url https://cep-weather-rwpinbrhjq-uc.a.run.app/weather/{cep}

## Tests

To run the tests, you need to update the [WEATHER_API_KEY] config on [.env] file inside the [/handler] path.
It is needed to execute get the data from WeatherAPI while testing the application.