# FlowC - Envoy xDS Control Plane
# Top-level Makefile that includes component-specific targets

include make/common.mk
include make/flowc.mk
include make/controller.mk

##@ Top-level Targets

.PHONY: all
all: flowc-build ## Build the FlowC control plane binary (default)

.PHONY: build
build: flowc-build ## Alias for flowc-build

.PHONY: run
run: flowc-run ## Alias for flowc-run

.PHONY: run-debug
run-debug: flowc-run-debug ## Alias for flowc-run-debug

.PHONY: test
test: test-all ## Alias for test-all

.PHONY: clean
clean: flowc-clean ## Clean all build artifacts
	rm -f coverage.out coverage.html
	rm -rf bin/

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
