PACKAGE := $(shell go list)
GOOS := $(shell go env GOOS)
GOARCH = $(shell go env GOARCH)
OBJ_DIR := $(GOPATH)/pkg/$(GOOS)_$(GOARCH)/$(PACKAGE)
GOLANGCILINT := v1.19.1

# Linter
.PHONY: lint-prepare
lint-prepare:
	@echo "Installing golangci-lint"
	@curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(go env GOPATH)/bin $(GOLANGCILINT)

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
	@if [ -d $(OBJ_DIR) ]; then \
		rm -rf $(OBJ_DIR); \
	fi
