PACKAGE := $(shell go list)
GOOS := $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
OBJ_DIR := $(GOPATH)/pkg/$(GOOS)_$(GOARCH)/$(PACKAGE)
GOLANGCILINT ?= v1.19.1

# Linter
.PHONY: lint-prepare
lint-prepare:
	@echo "Installing golangci-lint"
	@if [ ! $(GOLANGCILINT) == "none" ]; then \
		[ -d $(GOPATH)/bin ] || mkdir -p $(GOPATH)/bin; \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(GOLANGCILINT); \
	else \
		echo "Skipping due to GOLANGCILINT='none'"; \
	fi

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
	@go test $(TEST_OPTS)

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
	@[ -d $(OBJ_DIR) ] && rm -rf $(OBJ_DIR)
