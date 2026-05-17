.PHONY: build test lint vet staticcheck vuln install

GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null)
_VERSION_FILE := $(shell sed -n 's/^var Version = "\(.*\)"/\1/p' internal/version/version.go)
VERSION ?= $(_VERSION_FILE)
VERSION_PKG := github.com/hejunqiu/ds-code/internal/version
LDFLAGS := -X $(VERSION_PKG).Version=$(VERSION)
ifneq ($(GIT_COMMIT),)
LDFLAGS += -X main.gitCommit=$(GIT_COMMIT)
endif

build:
	go build -ldflags "$(LDFLAGS)" -o bin/ds-code ./cmd/ds-code

# Debug binary: /debug-panic and other dev-only hooks (see debug_panic_debug.go).
build-debug:
	go build -tags debug -ldflags "$(LDFLAGS)" -o bin/ds-code ./cmd/ds-code

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/ds-code

test:
	go test -race -count=1 ./...

test-integration:
	go test -tags=integration -race -count=1 ./internal/lsp/... ./internal/tool/builtin/...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

staticcheck:
	@command -v staticcheck >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
