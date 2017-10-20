# Dependencies Management
.PHONY: vendor-prepare
vendor-prepare:
	@echo "Installing dep"
	go get -u github.com/golang/dep/cmd/dep

dep.lock:
	@dep ensure -v -update

.PHONY: vendor-update
vendor-update:
	@dep ensure -v -update

vendor:
	@dep ensure -v

.PHONY: clean-vendor
clean-vendor:
	@rm -rf vendor

# Linter
.PHONY: lint-prepare
lint-prepare:
	@echo "Installing gometalinter"
	@go get -u github.com/alecthomas/gometalinter
	@gometalinter --install

.PHONY: lint
lint: vendor
	@gometalinter --cyclo-over=20 --deadline=2m $$(dep ensure -v)

# Testing
.PHONY: test
test: vendor
	@go test -bench=.

# Build and Installation
.PHONY: install
install: vendor
	@go install
