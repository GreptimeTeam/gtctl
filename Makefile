# Copyright 2023 Greptime Team
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

CLUSTER=e2e-cluster
LDFLAGS = $(shell ./hack/version.sh)

.PHONY: gtctl
gtctl: ## Build gtctl.
	@go build -ldflags '${LDFLAGS}' -o bin/gtctl ./cmd/gtctl

.PHONY: setup-e2e
setup-e2e: ## Setup e2e test environment.
	./hack/e2e/setup-e2e-env.sh

.PHONY: e2e
e2e: gtctl setup-e2e ## Run e2e.
	go test -timeout 8m -v ./tests/e2e/... && kind delete clusters ${CLUSTER}

.PHONY: test
test: ## Run unit test.
	go test -timeout 1m -v ./pkg/...

.PHONY: coverage
coverage: ## Run unit test with coverage.
	go test ./pkg/... -race -coverprofile=coverage.xml -covermode=atomic

.PHONY: fix-license-header
fix-license-header: license-eye ## Fix license header.
	license-eye -c .licenserc.yaml header fix

.PHONY: license-eye
license-eye: ## Install license-eye.
	@which license-eye || go install github.com/apache/skywalking-eyes/cmd/license-eye@latest

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# https://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display help messages.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
