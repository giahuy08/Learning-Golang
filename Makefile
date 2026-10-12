.PHONY: run fmt vet lint lint-fix test test-race test-cover bench tidy clean ci help

## run: run the scratch main.go (Phase 1 style playground)
run:
	go run main.go

## fmt: format code
fmt:
	go fmt ./...

## vet: run go vet
vet:
	go vet ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## lint-fix: run golangci-lint with autofix
lint-fix:
	golangci-lint run --fix ./...

## test: run all tests
test:
	go test -v -count=1 -timeout 60s ./...

## test-race: run tests with the race detector
test-race:
	go test -race -count=1 -timeout 60s ./...

## test-cover: run tests with coverage report
test-cover:
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## bench: run benchmarks
bench:
	go test -bench=. -benchmem ./...

## tidy: clean up go.mod / go.sum
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	@rm -rf bin coverage.out coverage.html
	@echo "Cleaned"

## ci: everything you'd run before committing
ci: fmt vet lint test-race

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
