# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: build

LDFLAGS = $(shell ./hack/version.sh)

build:
	@go build -ldflags '${LDFLAGS}' -o bin/gtctl ./cmd


.PHONY: ginkgo
ginkgo: ## install ginkgo
	go install github.com/onsi/ginkgo/v2/ginkgo@v2.5.1

.PHONY: sql-test
sql-test: ginkgo ## Run greptimedb sql tests
	$(GOBIN)/ginkgo -r ./tests
