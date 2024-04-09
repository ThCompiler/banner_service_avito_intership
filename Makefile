LOG_DIR=./logs
SWAG_DIRS=./internal/app/delivery/http/v1/,./internal/banner/delivery/http/v1/handlers,./internal/banner/delivery/http/v1/models/request,./internal/banner/delivery/http/v1/models/response,./external/auth/delivery/http/v1/handlers,./internal/app/delivery/http/tools
include ./config/env/api_test.env
export $(shell sed 's/=.*//' ./config/env/api_test.env)

# Запуск

.PHONY: run
run:
	sudo docker compose --env-file ./config/env/docker_run.env up -d

.PHONY: run-verbose
run-verbose:
	sudo docker compose --env-file ./config/env/docker_run.env up

.PHONY: stop
stop:
	sudo docker compose --env-file ./config/env/docker_run.env stop

.PHONY: down
down:
	sudo docker compose --env-file ./config/env/docker_run.env down

# Сборка

.PHONY: build
build:
	go build -o server -v ./cmd

.PHONY: swag-gen
swag-gen:
	swag init --parseDependency --parseInternal --parseDepth 1 -d $(SWAG_DIRS) -g ./swag_info.go -o docs

.PHONY: swag-fmt
swag-fmt:
	swag fmt -d $(SWAG_DIRS) -g ./swag_info.go

.PHONY: build-docker
build-docker:
	sudo docker build --no-cache --network host -f ./Dockerfile . --tag main

# Тест интеграции

.PHONY: run-environment
run-environment:
	sudo docker compose -f ./docker-compose-api-test.yml up -d

.PHONY: down-environment
down-environment:
	sudo docker compose -f ./docker-compose-api-test.yml down

.PHONY: run-api-test
run-api-test: run-environment
	sleep 5 # чтобы postgres успел инициализироваться
	go test -tags=integration ./...
	make down-environment

# Дополнительно

.PHONY: open-last-log
open-last-log:
	cat $(LOG_DIR)/`ls -t $(LOG_DIR) | head -1 `

.PHONY: clear-logs
clear-logs:
	rm -rf $(LOG_DIR)/*.log

.PHONY: mocks
mocks:
	go generate -n $$(go list ./internal/...)

.PHONY: run-coverage
run-coverage:
	go test -covermode=atomic -coverprofile=cover ./...
	cat cover | fgrep -v "mock" | fgrep -v "docs" | fgrep -v "config" > cover2
	go tool cover -func=cover2

.PHONY: fmt
fmt:
	gofumpt -e -w -extra .
	goimports -e -w .