# Copyright 2022 Greptime Team
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

.PHONY: gtctl
LDFLAGS = $(shell ./hack/version.sh)
gtctl:
	@go build -ldflags '${LDFLAGS}' -o bin/gtctl ./cmd

.PHONY: setup-e2e
setup-e2e: ## Setup e2e test environment.
	./hack/e2e/setup-e2e-env.sh

.PHONY: e2e
e2e: gtctl setup-e2e ## Run e2e
	go test -timeout 8m -v ./tests/e2e/... && kind delete clusters ${CLUSTER}
