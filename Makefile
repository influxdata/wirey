SHELL := /bin/bash

all: build

.PHONY: build
build:
	go build ./cmd/wirey

.PHONY: clean
clean:
	rm -Rf thirdy_party/
