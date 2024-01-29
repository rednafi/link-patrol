SHELL := /bin/bash

.PHONY: init
init:
	@echo "Initializing project"
	@go mod download
	@go mod tidy
	@cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -t -I{} go install {}


.PHONY: lint
lint:
	@echo "Running lint"
	@golines -w -m 92 cmd/* src/*
	@golangci-lint run --fix
	@go mod tidy
	@prettier --write .


.PHONY: lint-check
lint-check:
	@echo "Checking lint"
	@golangci-lint run


.PHONY: test
test:
	@echo "Running tests"
	@go test -v ./...


.PHONY: bench
bench:
	@echo "Running benchmarks"
	@go test -bench=. -benchmem ./...


.PHONY: clean
clean:
	@echo "Cleaning up"
	@go clean
	@rm -rf ./bin
