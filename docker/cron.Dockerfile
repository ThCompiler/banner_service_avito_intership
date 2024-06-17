FROM golang:1.22 as build
WORKDIR /app

COPY . .

RUN make build-cron

FROM golang:1.22 as production

WORKDIR /app

COPY --from=build /app/service .

ENV TASK_PERIOD 18000
ENV CONFIG_PATH /config.yaml

ENTRYPOINT /app/service --config=$CONFIG_PATH --period=$TASK_PERIOD