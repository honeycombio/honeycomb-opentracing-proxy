FROM golang:1.9-alpine

COPY . /go/src/github.com/honeycombio/zipkinproxy
WORKDIR /go/src/github.com/honeycombio/zipkinproxy
RUN go install ./...

FROM golang:1.9-alpine
COPY --from=0 /go/bin/zipkinproxy /go/bin/zipkinproxy
