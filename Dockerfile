FROM golang:1.9-alpine

COPY . /go/src/github.com/honeycombio/zipkinproxy
WORKDIR /go/src/github.com/honeycombio/zipkinproxy
RUN apk add --no-cache git
RUN go get ./...
RUN go install ./...
