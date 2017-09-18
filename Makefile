# Dependencies Management
.PHONY: vendor-prepare
vendor-prepare:
	@echo "Installing glide"
	@curl https://glide.sh/get | sh

glide.lock: glide.yaml
	@glide update

.PHONY: vendor-update
vendor-update:
	@glide update

vendor: glide.lock
	@glide install

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
	@gometalinter --cyclo-over=20 --deadline=2m $$(glide novendor)

# Testing
.PHONY: test
test: vendor
	@go test -bench=. $$(glide novendor)

# Build and Installation
.PHONY: install
install: vendor
	@go install $$(glide novendor)
