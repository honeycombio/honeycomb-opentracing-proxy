Zipkinproxy collects Zipkin span data and sends it to the Honeycomb API for
you to explore.

Zipkinproxy can also write span data to stdout for ease of local development,
and can transparently forward span data to another Zipkin collector.

Usage (subject to change):

```
# Forward spans to a Honeycomb dataset $DATASET, using writekey $WRITEKEY
zipkinproxy -d $DATASET -k $WRITEKEY

# Write spans to stdout
zipkinproxy --debug

# Forward spans to a downstream "real" Zipkin collector
zipkinproxy --downstream https://myzipkin.example.com:9411
```


This is a work in progress. Next steps:

- support sending spans to different datasets based on some criterion?
- more tests
- retry logic
- do something useful with annotations
- do something better with special-case binaryAnnotations
- support the Zipkin v2 API?
