## ── Configuration ────────────────────────────────────────────────────────────
BINARY      := bot
MODULE      := basics
MAIN        := ./cmd/bot
IMAGE       := $(MODULE)-bot
TAG         := latest

## ── Local build ──────────────────────────────────────────────────────────────

.PHONY: build
build: ## Compile the bot binary into ./bin/
	@mkdir -p bin
	go build -trimpath -ldflags="-s -w" -o bin/$(BINARY) $(MAIN)

.PHONY: run
run: ## Run the bot locally (requires .env with TELEGRAM_BOT_TOKEN and DATABASE_URL)
	go run $(MAIN)

.PHONY: migrate
migrate: ## Seed PostgreSQL with curated tests from data/topics.json (requires DATABASE_URL)
	go run ./cmd/migrate

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: vet
vet: ## Run go vet on all packages
	go vet ./...

.PHONY: tidy
tidy: ## Tidy and verify go.mod / go.sum
	go mod tidy
	go mod verify

.PHONY: clean
clean: ## Remove build artefacts
	rm -rf bin/

## ── Docker ───────────────────────────────────────────────────────────────────

.PHONY: docker-build
docker-build: ## Build the Docker image
	docker build -t $(IMAGE):$(TAG) .

.PHONY: docker-run
docker-run: ## Run the container (reads TELEGRAM_BOT_TOKEN from the shell env)
	@if [ -z "$$TELEGRAM_BOT_TOKEN" ]; then \
		echo "Error: TELEGRAM_BOT_TOKEN is not set in your environment."; \
		exit 1; \
	fi
	docker run --rm \
		-e TELEGRAM_BOT_TOKEN="$$TELEGRAM_BOT_TOKEN" \
		-e LOG_FORMAT=json \
		-e LOG_LEVEL=info \
		$(IMAGE):$(TAG)

.PHONY: docker-stop
docker-stop: ## Stop the running container (if any)
	-docker stop $$(docker ps -q --filter ancestor=$(IMAGE):$(TAG))

## ── Help ─────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
