FROM golang:latest as build
WORKDIR /app

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest

RUN make swag-gen
RUN make build

FROM golang:latest as production
WORKDIR /app

EXPOSE 8080

COPY --from=build /app/server .

RUN mkdir app-log

CMD ["./server"]