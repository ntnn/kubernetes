GO ?= go
GOLANGCILINT ?= golangci-lint

default: lint test

.PHONY: lint
lint:
	$(GOLANGCILINT) run ./...

.PHONY: test
test:
	$(GO) test -race ./...
