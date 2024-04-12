FROM golang:1.22 as build
WORKDIR /app

RUN go install github.com/swaggo/swag/cmd/swag@latest

COPY . .

RUN make swag-gen
RUN make build-banner

FROM golang:1.22 as production

WORKDIR /app

EXPOSE 8080

COPY --from=build /app/server .

RUN mkdir app-log

ENV CONFIG_PATH /config.yaml

ENTRYPOINT /app/server --config=$CONFIG_PATH