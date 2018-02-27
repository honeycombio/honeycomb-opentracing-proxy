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

If you're instrumenting a complex codebase, and you'd like to send different
types of traces to different Honeycomb datasets, add a `honeycomb.dataset` tag
to your spans. E.g.

```
span, ctx := opentracing.StartSpan("myNewSpan")
span.SetTag("honeycomb.dataset", "My Shiny Tracing Dataset")
```
