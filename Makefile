BIN := mysql-user-migrate

.PHONY: deps fmt lint test itest run

deps:
	go mod tidy

fmt:
	gofmt -w cmd internal
	@if command -v goimports >/dev/null 2>&1; then goimports -w cmd internal; fi

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not installed"; fi

test:
	go test ./...

itest:
	@if command -v docker-compose >/dev/null 2>&1; then docker-compose -f docker-compose.test.yml up --build --abort-on-container-exit; else echo "docker-compose not installed"; fi

run:
	go run ./cmd/$(BIN) $(ARGS)
