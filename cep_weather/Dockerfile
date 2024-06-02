FROM golang:1.21 as build
WORKDIR /app
COPY ./cep_weather /app
RUN CGO_ENABLED=0 GOOS=linux go build -o cep_weather
COPY ./cep_weather/.env /app

FROM scratch
WORKDIR /app
COPY --from=build /app/cep_weather .
COPY ./cep_weather/.env /app

ENTRYPOINT ["./cep_weather"]

#  # Use an official Golang runtime as a parent image
#  FROM golang:latest

#  # Set the working directory inside the container
#  WORKDIR /app/cmd

#  # Copy the local package files to the container's workspace
#  COPY . /app/cmd

#  # Build the Go application inside the container
#  RUN go build -o fullcycle_cep_weather

#  # Define the command to run your application
#  ENTRYPOINT ["./cep_weather"]
