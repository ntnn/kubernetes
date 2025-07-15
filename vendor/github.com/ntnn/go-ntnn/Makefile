GO ?= go
GOLANGCILINT ?= golangci-lint

default: lint

.PHONY: lint
lint:
	$(GOLANGCILINT) run ./...
