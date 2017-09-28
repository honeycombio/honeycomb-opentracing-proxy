Extremely proof-of-concept-level zipkin-to-Honeycomb forwarder. Speaks the
zipkin V1 API: send your spans to `/api/v1/spans`.

Usage (subject to change):

```
# Forward spans to a Honeycomb dataset $DATASET, using writekey $WRITEKEY
zipkinproxy -d $DATASET -k $WRITEKEY

# Write spans to stdout
zipkinproxy --debug

# Forward spans to an upstream "real" Zipkin collector
zipkinproxy --upstream https://myzipkin.example.com:9411
```


Next steps:

- more tests
- retry logic
- do something useful with annotations
- do something better with special-case binaryAnnotations

