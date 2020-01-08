FROM golang:1.13-alpine

COPY . /go/src/github.com/honeycombio/honeycomb-opentracing-proxy
WORKDIR /go/src/github.com/honeycombio/honeycomb-opentracing-proxy
RUN go install -mod=vendor ./...

FROM golang:1.13-alpine
COPY --from=0 /go/bin/honeycomb-opentracing-proxy /honeycomb-opentracing-proxy
ENTRYPOINT ["/honeycomb-opentracing-proxy"]
