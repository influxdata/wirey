SHELL := /bin/bash
COMMIT_NO := $(shell git rev-parse --short=7 HEAD 2> /dev/null || true)
GIT_COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")
LDFLAGS=-ldflags "-s -X main.Version=${GIT_COMMIT}"

.PHONY: help
help: ## This is the help target
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

all: build ## Build wirey for local development

.PHONY: clean
clean: ## Cleanup build directories
	$(RM) -r bin/
	$(RM) -r dist/

.PHONY: build
build: ## Build wirey for local development
	mkdir -p bin
	go build ${LDFLAGS} -o bin/wirey ./cmd/wirey

.PHONY: vendor
vendor: ## Execute go mod vendor
	go mod vendor

.PHONY: test
test: vendor ## Execute go test
	go test -v ./...

.PHONY: fmt
fmt: ## Execute go fmt
	go fmt ./...

.PHONY: tidy
tidy: ## Execute go mod tidy
	go mod tidy


gorelease: ## Execute goreleaser with snapshot flag
	goreleaser release --snapshot --rm-dist
