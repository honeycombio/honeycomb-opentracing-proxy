FROM golang:1.9-alpine

COPY . /go/src/github.com/honeycombio/honeycomb-opentracing-proxy
WORKDIR /go/src/github.com/honeycombio/honeycomb-opentracing-proxy
RUN go install ./...

FROM golang:1.9-alpine
COPY --from=0 /go/bin/honeycomb-opentracing-proxy /honeycomb-opentracing-proxy
ENTRYPOINT ["/honeycomb-opentracing-proxy"]
