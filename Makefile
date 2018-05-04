SHELL := /bin/bash
COMMIT_NO := $(shell git rev-parse --short=7 HEAD 2> /dev/null || true)
GIT_COMMIT := $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")
LDFLAGS=-ldflags "-s -X main.Version=${GIT_COMMIT}"

all: build

.PHONY: build
build:
	mkdir -p bin
	go build ${LDFLAGS} -o bin/wirey ./cmd/wirey

