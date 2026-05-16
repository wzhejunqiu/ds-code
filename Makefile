.PHONY: build test lint vet vuln install

build:
	go build -o bin/ds-code ./cmd/ds-code

install:
	go install ./cmd/ds-code

test:
	go test -race -count=1 ./...

test-integration:
	go test -tags=integration -race -count=1 ./internal/lsp/... ./internal/tool/builtin/...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
