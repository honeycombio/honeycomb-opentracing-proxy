`honeycomb-opentracing-proxy` is a drop-in compatible replacement for Zipkin.
If your services are instrumented with OpenTracing and emit span data using
Zipkin's wire format, then `honeycomb-opentracing-proxy` can receive that data
and forward it to the [Honeycomb](https://honeycomb.io) API. Using Honeycomb,
you can explore single traces, and run queries over aggregated trace data.

## Usage

```
# Forward spans to a Honeycomb dataset $DATASET, using writekey $WRITEKEY
honeycomb-opentracing-proxy -d $DATASET -k $WRITEKEY

# Write spans to stdout
honeycomb-opentracing-proxy --debug

# Forward spans to a downstream "real" Zipkin collector
honeycomb-opentracing-proxy --downstream https://myzipkin.example.com:9411
```


This is a work in progress. Next steps:

- support sending spans to different datasets based on some criterion?
- more tests
- retry logic
- do something useful with annotations
- do something better with special-case binaryAnnotations
- support the Zipkin v2 API?
