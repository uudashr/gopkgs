PACKAGE := $(shell go list)
GOOS := $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
OBJ_DIR := $(GOPATH)/pkg/$(GOOS)_$(GOARCH)/$(PACKAGE)
GO_VERSION_MINOR := $(shell go version | sed -E 's/.*go([0-9]\.[0-9]+).*/\1/g')
TRAVIS_GO_VERSION ?= $(GO_VERSION_MINOR).x

GOLANGCI_LINT_1.9.x := v1.10.2
GOLANGCI_LINT_1.10.x := v1.15.0
GOLANGCI_LINT_1.11.x := v1.17.1
GOLANGCI_LINT_1.12.x := v1.21.0
GOLANGCI_LINT_1.13.x := v1.21.0
GOLANGCI_LINT_tip := v1.21.0
GOLANGCI_LINT := ${GOLANGCI_LINT_${TRAVIS_GO_VERSION}}

# Linter
.PHONY: lint-prepare
lint-prepare:
	@if [ $(GOLANGCI_LINT) == "" ]; then \
		echo "Unknown Go version - using the latest linter"; \
		GOLANGCI_LINT := $(GOLANGCI_LINT_tip); \
	fi
	@echo "Installing golangci-lint $(GOLANGCI_LINT)"
	@[ -d $(GOPATH)/bin ] || mkdir -p $(GOPATH)/bin
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCI_LINT)

.PHONY: lint
lint: 
	@golangci-lint run \
		--exclude-use-default=false \
		--enable=golint \
		--enable=gocyclo \
		--enable=goconst \
		--enable=unconvert \
		--exclude='^Error return value of `.*\.Log` is not checked$$' \
		--exclude='^G104: Errors unhandled\.$$' \
		--exclude='^G304: Potential file inclusion via variable$$' \
		./...

# Testing
.PHONY: test
test: 
	@go test $(TEST_OPTS) github.com/uudashr/gopkgs/v2

.PHONY: bench
bench: 
	@go test -run=none -bench=. -benchmem

# Build and Installation
.PHONY: install
install:
	@go install ./...

.PHONY: uninstall
uninstall:
	@echo "Removing binaries and libraries"
	@go clean -i ./...
	@if [ -d $(OBJ_DIR) ]; then rm -rf $(OBJ_DIR); fi
