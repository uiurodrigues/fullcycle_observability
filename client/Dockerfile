FROM golang:1.21 as build
WORKDIR /app
COPY ./client /app
RUN CGO_ENABLED=0 GOOS=linux go build -o cep_client

FROM scratch
WORKDIR /app
COPY --from=build /app/cep_client .

ENTRYPOINT ["./cep_client"]