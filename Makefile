SHELL := /bin/bash
COMMIT_NO := $(shell git rev-parse --short=7 HEAD 2> /dev/null || true)
GIT_COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")
LDFLAGS=-ldflags "-s -X main.Version=${GIT_COMMIT}"

all: build

.PHONY: clean
clean:
	$(RM) -r bin/

.PHONY: build
build:
	mkdir -p bin
	go build ${LDFLAGS} -o bin/wirey ./cmd/wirey

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: test
test: vendor
	go test -v ./...

.PHONY: fmt
fmt:
	go fmt ./...


gorelease:
	goreleaser release --snapshot --rm-dist
