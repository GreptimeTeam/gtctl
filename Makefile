.PHONY: gtctl

LDFLAGS = $(shell ./hack/version.sh)

gtctl:
	@go build -ldflags '${LDFLAGS}' -o bin/gtctl ./cmd

github-release: gtctl
	mv bin/* .

check-format:
fmt-check: ## Check files format.
	echo "Checking files format ..."
	go fmt ./... | grep . && { echo "Unformatted files found"; exit 1; } || echo "No file to format"
