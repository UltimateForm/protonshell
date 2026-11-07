BINARY  := protonshell
PREFIX  ?= $(HOME)/.local
BINDIR  := $(PREFIX)/bin
GOFLAGS ?= -trimpath
LDFLAGS ?= -s -w
ARGS    ?=
OUT     := .out
BIN     := $(OUT)/$(BINARY)

.DEFAULT_GOAL := build

.PHONY: build run install uninstall test vet fmt fmt-check tidy check clean help

build: ## Build the binary
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BIN) .

run: build ## Build and run (pass flags with ARGS="-iface wg0")
	$(BIN) $(ARGS)

install: build ## Install to $(BINDIR)
	install -Dm755 $(BIN) $(BINDIR)/$(BINARY)

uninstall: ## Remove from $(BINDIR)
	rm -f $(BINDIR)/$(BINARY)

test: ## Run tests
	go test ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format sources
	gofmt -w .

fmt-check: ## Fail if any file needs formatting
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "needs gofmt:"; echo "$$out"; exit 1; fi

tidy: ## Tidy go.mod/go.sum
	go mod tidy

check: fmt-check vet test ## Run all checks

clean: ## Remove build output
	rm -rf $(OUT)

help: ## List targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-10s %s\n", $$1, $$2}'
