.PHONY: gtctl

LDFLAGS = $(shell ./hack/version.sh)

gtctl:
	@go build -ldflags '${LDFLAGS}' -o bin/gtctl ./cmd

github-release: gtctl
	mv bin/* .
