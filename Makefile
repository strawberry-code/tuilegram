BINARY_NAME := tuilegram
MAIN_PKG := ./cmd/tuilegram

GOLANGCI_LINT_VERSION := v1.62.2

.PHONY: build run rerun test lint lint-install vet install clean tidy loc-check

build:
	go build -o $(BINARY_NAME) $(MAIN_PKG)

run:
	go run $(MAIN_PKG)

rerun:
	rm -f session.json
	go run $(MAIN_PKG)

test:
	go test -v -count=1 ./...

vet:
	go vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint non trovato. Installa con: make lint-install"; \
		echo "Fallback → go vet ./..."; \
		go vet ./...; \
	fi

lint-install:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo "golangci-lint installato in $$(go env GOPATH)/bin"
	@echo "Aggiungi $$(go env GOPATH)/bin al PATH se non già presente."

install:
	go install $(MAIN_PKG)

tidy:
	go mod tidy

clean:
	rm -f $(BINARY_NAME)

loc-check:
	@find internal cmd -name '*.go' -type f -exec wc -l {} + \
	  | awk '$$1>120 && $$2!="total"' \
	  | sort -nr \
	  | (grep . && exit 1) || echo "loc-check: all Go files within 120 LOC."
