FROM golang:1.12.9-alpine3.10 as build

# Set the Current Working Directory inside the container
WORKDIR /app

# Install dependencies
RUN apk update && apk upgrade && apk add --no-cache \
  bash \ 
  git \
  bzr

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
ENV GOOS linux
ENV GARCH amd64
ENV CGO_ENABLED 0

RUN go build -ldflags="-s -w" -o wirey cmd/wirey/main.go

# Release container
FROM alpine:3.10

WORKDIR /app

COPY --from=build /app/wirey /app/wirey

# Command to run the executable
ENTRYPOINT ["/app/wirey"]
