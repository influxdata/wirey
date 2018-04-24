SHELL := /bin/bash

all: build

.PHONY: build
build:
	mkdir -p bin
	go build -o bin/wirey ./cmd/wirey

