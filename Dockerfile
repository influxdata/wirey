FROM golang:1.10.1-alpine3.7 as builder

ENV DEP_VERSION 0.4.1
RUN apk -U add openssl curl git
RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 && chmod +x /usr/local/bin/dep

ADD . /go/src/github.com/influxdata/wirey
WORKDIR /go/src/github.com/influxdata/wirey

RUN dep ensure -vendor-only

RUN go install ./cmd/wirey

WORKDIR /go/src/github.com/influxdata/wirey
RUN go get ./...
RUN go install ./...


FROM alpine:3.7
COPY --from=builder /go/bin/wirey /usr/local/bin
COPY --from=builder /go/bin/httpbackend /usr/local/bin

ENTRYPOINT ["/usr/local/bin/wirey"]
