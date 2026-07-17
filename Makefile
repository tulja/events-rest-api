BINARY := events-rest-api
GO     ?= go

.PHONY: help build run test test-v fmt tidy clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?##"}; {printf "  %-12s %s\n", $$1, $$2}'

build: ## Build binary to ./events-rest-api
	$(GO) build -o $(BINARY) .

run: ## Run the API (JWT_SIGNING_KEY must be set)
	$(GO) run .

test: ## Run all tests once
	$(GO) test ./... -count=1

test-v: ## Run all tests once with verbose output
	$(GO) test ./... -count=1 -v

fmt: ## Format Go sources
	$(GO) fmt ./...

tidy: ## Tidy go.mod / go.sum
	$(GO) mod tidy

clean: ## Remove built binary
	rm -f $(BINARY)
