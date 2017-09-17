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

# Testing
.PHONY: test
test: vendor
	@go test -bench=. $$(glide novendor)
