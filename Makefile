.PHONY: build build-tui-test test test-tui test-integration cover cover-html verify-release verify-charm-v2 lint vet staticcheck vuln install fetch-tokenizers fetch-ripgrep check-commit check-push install-hooks

COVERPROFILE ?= coverage.out

TOKENIZERS_LIB := third_party/tokenizers/libtokenizers.a
RIPGREP_TAR := internal/tool/builtin/grep/rgbin/rg.tar.gz

$(TOKENIZERS_LIB): scripts/fetch-tokenizers-lib.sh
	./scripts/fetch-tokenizers-lib.sh

$(RIPGREP_TAR): scripts/fetch-ripgrep.sh
	./scripts/fetch-ripgrep.sh

fetch-tokenizers: $(TOKENIZERS_LIB)

fetch-ripgrep: $(RIPGREP_TAR)

GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null)
_VERSION_FILE := $(shell sed -n 's/^var Version = "\(.*\)"/\1/p' internal/version/version.go)
VERSION ?= $(_VERSION_FILE)
VERSION_PKG := github.com/wzhejunqiu/ds-code/internal/version
# Release builds override Version via GitHub Release workflow ldflags.
LDFLAGS := -X $(VERSION_PKG).Version=$(VERSION)
ifneq ($(GIT_COMMIT),)
LDFLAGS += -X main.gitCommit=$(GIT_COMMIT)
endif

build: $(TOKENIZERS_LIB) $(RIPGREP_TAR)
	go build -ldflags "$(LDFLAGS)" -o bin/ds-code ./cmd/ds-code

# Debug binary: /debug-panic and other dev-only hooks (see debug_panic_debug.go).
build-debug: $(TOKENIZERS_LIB) $(RIPGREP_TAR)
	go build -tags debug -ldflags "$(LDFLAGS)" -o bin/ds-code ./cmd/ds-code

install: $(TOKENIZERS_LIB) $(RIPGREP_TAR)
	go install -ldflags "$(LDFLAGS)" ./cmd/ds-code

test: $(TOKENIZERS_LIB) $(RIPGREP_TAR)
	go test -race -count=1 ./...

cover: $(TOKENIZERS_LIB)
	go test -race -count=1 -coverprofile=$(COVERPROFILE) ./...
	go tool cover -func=$(COVERPROFILE) | tail -1

cover-html: cover
	go tool cover -html=$(COVERPROFILE) -o coverage.html
	@echo "wrote coverage.html"

build-tui-test: $(TOKENIZERS_LIB)
	go build -tags=tuitest -ldflags "$(LDFLAGS)" -o bin/ds-code-tui-test ./cmd/ds-code-tui-test

test-tui: $(TOKENIZERS_LIB)
	go test -tags=tuitest -race -count=1 ./internal/tuitest/...

verify-charm-v2:
	@! rg 'github.com/charmbracelet/(bubbletea|bubbles|lipgloss|glamour)' \
		--glob '*.go' --glob 'go.mod' . | grep -v '^#' \
		|| (echo "v1 charm import detected"; exit 1)
	@rg -q 'charm.land/bubbletea/v2' go.mod || (echo "missing charm.land/bubbletea/v2"; exit 1)
	@rg -q 'charm.land/bubbles/v2' go.mod || (echo "missing charm.land/bubbles/v2"; exit 1)
	@rg -q 'charm.land/lipgloss/v2' go.mod || (echo "missing charm.land/lipgloss/v2"; exit 1)
	@rg -q 'charm.land/glamour/v2' go.mod || (echo "missing charm.land/glamour/v2"; exit 1)

verify-release: $(TOKENIZERS_LIB) verify-charm-v2
	go build -ldflags "$(LDFLAGS)" -o bin/ds-code ./cmd/ds-code
	@! strings bin/ds-code | grep -qE '/tcase|tuitest|__tcase__' || (echo "release binary contains tuitest strings"; exit 1)

test-integration: $(TOKENIZERS_LIB)
	go test -tags=integration -race -count=1 ./internal/lsp/... ./internal/tool/builtin/...

vet: $(TOKENIZERS_LIB)
	go vet -copylocks ./...

GOLANGCI_LINT_VERSION := $(shell cat .golangci-lint-version)
GOPATH_BIN := $(shell go env GOPATH)/bin

lint: $(TOKENIZERS_LIB)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(GOPATH_BIN)/golangci-lint run ./...

staticcheck: $(TOKENIZERS_LIB)
	@command -v staticcheck >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck ./...

vuln: $(TOKENIZERS_LIB)
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

check-commit: $(TOKENIZERS_LIB)
	./scripts/pre-commit-check.sh
	$(MAKE) vuln

check-push: $(TOKENIZERS_LIB)
	./scripts/check-gofmt.sh
	$(MAKE) vet lint vuln

install-hooks:
	git config --local core.hooksPath .githooks
	chmod +x .githooks/pre-commit .githooks/pre-push \
		scripts/check-gofmt.sh scripts/pre-commit-check.sh scripts/lint-packages.sh
	@echo "git hooks installed (.githooks → check-commit / check-push, includes vuln)"
