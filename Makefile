LOG_DIR=./logs
SWAG_DIRS=./internal/app/delivery/http/v1/,./internal/banner/delivery/http/v1/handlers,./internal/banner/delivery/http/v1/models/request,./internal/banner/delivery/http/v1/models/response,./external/auth/delivery/http/v1/handlers,./internal/app/delivery/http/tools
include ./config/env/api_test.env
export $(shell sed 's/=.*//' ./config/env/api_test.env)

# Запуск

.PHONY: create-nginx-logs
create-nginx-logs:
	mkdir -p logs-nginx

.PHONY: run
run: create-nginx-logs
	sudo docker compose --env-file ./config/env/docker_run.env up -d

.PHONY: run-verbose
run-verbose: create-nginx-logs
	sudo docker compose --env-file ./config/env/docker_run.env up

.PHONY: stop
stop:
	sudo docker compose --env-file ./config/env/docker_run.env stop

.PHONY: down
down:
	sudo docker compose --env-file ./config/env/docker_run.env down

# Сборка

.PHONY: build-banner
build-banner:
	go build -o server -v ./cmd/banner

.PHONY: build-cron
build-cron:
	go build -o service -v ./cmd/cron

.PHONY: build
build: build-cron build-banner


.PHONY: swag-gen
swag-gen:
	swag init --parseDependency --parseInternal --parseDepth 1 -d $(SWAG_DIRS) -g ./swag_info.go -o docs

.PHONY: swag-fmt
swag-fmt:
	swag fmt -d $(SWAG_DIRS) -g ./swag_info.go

.PHONY: build-docker-banner
build-docker-banner:
	sudo docker build --no-cache --network host -f ./docker/banner.Dockerfile . --tag banner

.PHONY: build-docker-cron
build-docker-cron:
	sudo docker build --no-cache --network host -f ./docker/cron.Dockerfile . --tag cron

.PHONY: build-docker-all
build-docker-all: build-docker-cron build-docker-banner

# Тест интеграции

.PHONY: run-environment-with-build
run-environment-with-build: build-docker-cron
	sudo docker compose -f ./docker-compose-api-test.yml up -d

.PHONY: run-environment
run-environment:
	sudo docker compose -f ./docker-compose-api-test.yml up -d

.PHONY: down-environment
down-environment:
	sudo docker compose -f ./docker-compose-api-test.yml down

.PHONY: run-api-test
run-api-test:
	go test -tags=integration ./...

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